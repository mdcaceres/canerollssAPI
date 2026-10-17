package handlers

import (
	"canerollss/adapters/input/http/session"
	"canerollss/adapters/output/whatsapp"
	"canerollss/core/usecase"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type WhatsAppHandler struct {
	agentUseCase *usecase.WhatsAppAgentUseCase
	sessionMgr   *session.SessionManager
	waClient     *whatsapp.WhatsAppClient
	verifyToken  string
	processedMu  sync.Mutex
	processedIDs map[string]time.Time
}

func NewWhatsAppHandler(
	agent *usecase.WhatsAppAgentUseCase,
	client *whatsapp.WhatsAppClient,
) *WhatsAppHandler {
	verifyToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
	if verifyToken == "" {
		verifyToken = "canerollss_verify_secret"
	}

	h := &WhatsAppHandler{
		agentUseCase: agent,
		waClient:     client,
		verifyToken:  verifyToken,
		processedIDs: make(map[string]time.Time),
	}

	// Initialize session manager with 4-second debounce timer
	h.sessionMgr = session.NewSessionManager(4*time.Second, h.handleConsolidatedMessage)

	// Cleaner for processed message IDs
	go h.startIdCleaner()

	return h
}

// VerifyWebhook handles GET /webhook/whatsapp
func (h *WhatsAppHandler) VerifyWebhook(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.verifyToken {
		log.Println("[WhatsApp Webhook] Verification successful")
		return c.Status(fiber.StatusOK).SendString(challenge)
	}

	log.Printf("[WhatsApp Webhook] Verification failed: mode=%s, token=%s", mode, token)
	return c.Status(fiber.StatusForbidden).SendString("Forbidden")
}

type metaWebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Statuses []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// HandleWebhook handles POST /webhook/whatsapp
func (h *WhatsAppHandler) HandleWebhook(c *fiber.Ctx) error {
	raw := c.Body()
	body := make([]byte, len(raw))
	copy(body, raw)

	// Acknowledge Meta immediately (< 2s) with HTTP 200
	c.Status(fiber.StatusOK).SendString("EVENT_RECEIVED")

	// Process message asynchronously
	go h.processAsyncWebhook(body)

	return nil
}

func (h *WhatsAppHandler) processAsyncWebhook(body []byte) {
	var payload metaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[WhatsApp Webhook] JSON parse error: %v", err)
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			val := change.Value

			// Ignore status updates (sent, delivered, read)
			if len(val.Statuses) > 0 && len(val.Messages) == 0 {
				continue
			}

			// Map profile names
			profileName := "Cliente"
			if len(val.Contacts) > 0 && val.Contacts[0].Profile.Name != "" {
				profileName = val.Contacts[0].Profile.Name
			}

			for _, msg := range val.Messages {
				// Deduplication check
				if h.isAlreadyProcessed(msg.ID) {
					log.Printf("[WhatsApp Webhook] Duplicate message skipped: %s", msg.ID)
					continue
				}
				h.markProcessed(msg.ID)

				if msg.Type == "text" && msg.Text.Body != "" {
					h.sessionMgr.AddMessage(msg.From, profileName, msg.Text.Body)
				}
			}
		}
	}
}

func (h *WhatsAppHandler) handleConsolidatedMessage(phone, profileName, combinedText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	history := h.sessionMgr.GetHistory(phone)
	sessionCtx := h.agentUseCase.BuildSessionContext(phone, profileName)

	reply, updatedHistory, err := h.agentUseCase.ProcessMessage(ctx, sessionCtx, combinedText, history)
	if err != nil {
		log.Printf("[WhatsApp Webhook] Error processing message from %s: %v", phone, err)
		reply = "Disculpá, tuvimos un inconveniente momentáneo al procesar tu solicitud. ¿Podrías intentar de nuevo?"
	}

	if updatedHistory != nil {
		h.sessionMgr.SetHistory(phone, updatedHistory)
	}

	// Send outbound reply to user
	if err := h.waClient.SendTextMessage(ctx, phone, reply); err != nil {
		log.Printf("[WhatsApp Webhook] Error sending outbound message to %s: %v", phone, err)
	}
}

func (h *WhatsAppHandler) isAlreadyProcessed(id string) bool {
	h.processedMu.Lock()
	defer h.processedMu.Unlock()
	_, exists := h.processedIDs[id]
	return exists
}

func (h *WhatsAppHandler) markProcessed(id string) {
	h.processedMu.Lock()
	defer h.processedMu.Unlock()
	h.processedIDs[id] = time.Now()
}

func (h *WhatsAppHandler) startIdCleaner() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.processedMu.Lock()
		cutoff := time.Now().Add(-1 * time.Hour)
		for id, t := range h.processedIDs {
			if t.Before(cutoff) {
				delete(h.processedIDs, id)
			}
		}
		h.processedMu.Unlock()
	}
}

type SimulateChatRequest struct {
	Phone   string `json:"phone"`
	Name    string `json:"name"`
	Message string `json:"message"`
	Reset   bool   `json:"reset"`
}

// SimulateChat provides a synchronous, multi-turn testing endpoint for Postman and curl
func (h *WhatsAppHandler) SimulateChat(c *fiber.Ctx) error {
	var req SimulateChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "JSON inválido en el cuerpo de la solicitud",
		})
	}

	if strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El campo 'message' es obligatorio",
		})
	}

	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		phone = "5493519876543"
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Matias"
	}

	if req.Reset {
		h.sessionMgr.SetHistory(phone, nil)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 35*time.Second)
	defer cancel()

	history := h.sessionMgr.GetHistory(phone)
	sessionCtx := h.agentUseCase.BuildSessionContext(phone, name)

	reply, updatedHistory, err := h.agentUseCase.ProcessMessage(ctx, sessionCtx, req.Message, history)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error procesando mensaje: %v", err),
		})
	}

	if updatedHistory != nil {
		h.sessionMgr.SetHistory(phone, updatedHistory)
	}

	return c.JSON(fiber.Map{
		"reply": reply,
		"phone": phone,
		"name":  name,
		"session": fiber.Map{
			"current_time":        sessionCtx.CurrentTime,
			"cash_register_open": sessionCtx.IsCashRegisterOpen,
			"has_saved_addresses": sessionCtx.HasSavedAddresses,
			"pending_order_id":    sessionCtx.PendingOrderID,
		},
	})
}
