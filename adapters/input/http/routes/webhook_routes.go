package routes

import (
	"canerollss/adapters/input/http/handlers"

	"github.com/gofiber/fiber/v2"
)

func MapWebhookRoutes(app *fiber.App, waH *handlers.WhatsAppHandler, mpH *handlers.MercadoPagoHandler, uber ...*handlers.UberDeliveryWebhookHandler) {
	webhookGroup := app.Group("/webhook")

	// WhatsApp Cloud API
	webhookGroup.Get("/whatsapp", waH.VerifyWebhook)
	webhookGroup.Post("/whatsapp", waH.HandleWebhook)

	// Simulation endpoint for Postman / curl testing
	webhookGroup.Post("/whatsapp/simulate", waH.SimulateChat)

	// Direct friendly route for Postman / curl
	app.Post("/chat", waH.SimulateChat)

	// Mercado Pago IPN
	webhookGroup.Post("/mercadopago", mpH.HandleWebhook)
	if len(uber) > 0 && uber[0] != nil {
		webhookGroup.Post("/uber-delivery", uber[0].HandleWebhook)
	}
}
