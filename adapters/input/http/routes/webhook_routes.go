package routes

import (
	"canerollss/adapters/input/http/handlers"

	"github.com/gofiber/fiber/v2"
)

func MapWebhookRoutes(app *fiber.App, waH *handlers.WhatsAppHandler, mpH *handlers.MercadoPagoHandler) {
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
}
