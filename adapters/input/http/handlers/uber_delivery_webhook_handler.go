package handlers

import (
	"canerollss/core/usecase"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"strings"
)

type UberDeliveryWebhookHandler struct {
	delivery *usecase.DeliveryUseCase
	secret   string
}

func NewUberDeliveryWebhookHandler(delivery *usecase.DeliveryUseCase, secret string) *UberDeliveryWebhookHandler {
	return &UberDeliveryWebhookHandler{delivery: delivery, secret: secret}
}
func (h *UberDeliveryWebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	if h.secret == "" {
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	signature := strings.TrimSpace(c.Get("X-Uber-Signature"))
	provided, err := hex.DecodeString(signature)
	if err != nil || len(provided) != sha256.Size {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(c.Body())
	if subtle.ConstantTimeCompare(provided, mac.Sum(nil)) != 1 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var event struct {
		Kind       string `json:"kind"`
		DeliveryID string `json:"delivery_id"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(c.Body(), &event); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	if event.Kind != "event.delivery_status" || event.DeliveryID == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	if err := h.delivery.ApplyCourierStatus(event.DeliveryID, event.Status); err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	return c.SendStatus(fiber.StatusOK)
}
