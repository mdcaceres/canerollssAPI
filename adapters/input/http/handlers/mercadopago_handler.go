package handlers

import (
	"canerollss/adapters/output/whatsapp"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"canerollss/ports/input"
	"canerollss/ports/output"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MercadoPagoHandler struct {
	orderUseCase     *usecase.OrderUseCase
	paymentPort      output.PaymentProvider
	reconciliationUC input.PaymentReconciliationUseCasePort
	waClient         *whatsapp.WhatsAppClient
	webhookSecret    string
}

func NewMercadoPagoHandler(
	orderUC *usecase.OrderUseCase,
	payment output.PaymentProvider,
	reconciliationUC input.PaymentReconciliationUseCasePort,
	waClient *whatsapp.WhatsAppClient,
	webhookSecret string,
) *MercadoPagoHandler {
	return &MercadoPagoHandler{
		orderUseCase:     orderUC,
		paymentPort:      payment,
		reconciliationUC: reconciliationUC,
		waClient:         waClient,
		webhookSecret:    webhookSecret,
	}
}

type mpWebhookNotification struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	Topic  string `json:"topic"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (h *MercadoPagoHandler) HandleWebhook(c *fiber.Ctx) error {
	var payload mpWebhookNotification
	_ = c.BodyParser(&payload)

	paymentID := payload.Data.ID
	if paymentID == "" {
		paymentID = c.Query("data.id")
		if paymentID == "" {
			paymentID = c.Query("id")
		}
	}

	if paymentID == "" {
		return c.Status(fiber.StatusOK).SendString("IGNORED")
	}

	// HMAC-SHA256 signature verification if secret is configured
	if h.webhookSecret != "" {
		sigHeader := c.Get("x-signature")
		if sigHeader == "" {
			sigHeader = c.Query("x-signature")
		}
		reqID := c.Get("x-request-id")
		if !VerifyHMACSignature(h.webhookSecret, sigHeader, reqID, paymentID) {
			log.Printf("[MercadoPago Webhook] Invalid HMAC signature for payment %s", paymentID)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid webhook signature"})
		}
	}

	go func() {
		_, _ = h.ProcessPaymentUpdateSync(context.Background(), paymentID)
	}()

	return c.Status(fiber.StatusOK).SendString("OK")
}

// ProcessPaymentUpdateSync performs reconciliation synchronously (useful for webhooks and testing)
func (h *MercadoPagoHandler) ProcessPaymentUpdateSync(ctx context.Context, paymentID string) (*domain.ReconciliationResult, error) {
	// 1. First attempt transfer reconciliation through the reconciliation usecase
	if h.reconciliationUC != nil {
		reconRes, err := h.reconciliationUC.ReconcileIncomingTransfer(ctx, paymentID)
		if err == nil && reconRes != nil && reconRes.Status == domain.ReconciliationStatusMatched {
			log.Printf("[MercadoPago Webhook] Transfer %s successfully reconciled to order #%d", paymentID, reconRes.OrderID)
			if h.orderUseCase != nil {
				order, _ := h.orderUseCase.GetOrderByID(reconRes.OrderID)
				if order != nil && order.Customer.Whatsapp != "" && h.waClient != nil {
					msg := fmt.Sprintf("¡Hola %s! 🤎 Recibimos tu transferencia por $%.2f para el pedido #%d. ¡Tu pedido ya ingresó a cocina para preparación y despacho!\n\nTus rolls salen recién horneados y listos para disfrutar ✨", order.Customer.Name, reconRes.MatchedTransaction.Amount, order.ID)
					_ = h.waClient.SendTextMessage(ctx, order.Customer.Whatsapp, msg)
				}
			}
			return reconRes, nil
		}
	}

	// 2. Fallback to legacy payment provider check if external_reference or direct payment ID
	if h.paymentPort != nil {
		result, err := h.paymentPort.VerifyPaymentStatus(ctx, paymentID)
		if err != nil {
			log.Printf("[MercadoPago Webhook] Error checking payment %s: %v", paymentID, err)
			return nil, err
		}

		if result.IsPaid {
			log.Printf("[MercadoPago Webhook] Payment %s approved!", paymentID)
			if orderID, err := strconv.ParseUint(paymentID, 10, 32); err == nil && h.orderUseCase != nil {
				_ = h.orderUseCase.UpdateOrderStatus(uint(orderID), domain.OrderStatusPaid)
				order, _ := h.orderUseCase.GetOrderByID(uint(orderID))
				if order != nil && order.Customer.Whatsapp != "" && h.waClient != nil {
					msg := fmt.Sprintf("¡Hola %s! Tu pago para el pedido #%d ha sido confirmado con éxito. Ya estamos preparando tus roles de canela recién horneados.", order.Customer.Name, order.ID)
					_ = h.waClient.SendTextMessage(ctx, order.Customer.Whatsapp, msg)
				}
			}
		}
	}

	return nil, nil
}

// VerifyHMACSignature validates the x-signature header from Mercado Pago
func VerifyHMACSignature(secret, signatureHeader, requestID, dataID string) bool {
	if secret == "" {
		return true
	}
	if signatureHeader == "" {
		return false
	}

	var ts, v1 string
	parts := strings.Split(signatureHeader, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			switch strings.TrimSpace(kv[0]) {
			case "ts":
				ts = strings.TrimSpace(kv[1])
			case "v1":
				v1 = strings.TrimSpace(kv[1])
			}
		}
	}

	if ts == "" || v1 == "" {
		return false
	}

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, requestID, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(v1)))
}
