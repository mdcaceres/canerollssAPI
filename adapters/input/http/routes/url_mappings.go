package routes

import (
	"canerollss/adapters/input/http/handlers"
	"canerollss/adapters/input/http/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func MapUrls(app *fiber.App, authH *handlers.AuthHandler, cashH *handlers.CashHandler, catalogH *handlers.CatalogHandler, orderH *handlers.OrderHandler, reportH *handlers.ReportHandler, customerH *handlers.CustomerHandler) {
	micro := app.Group("/api")

	micro.Post("/auth/register", authH.Register)
	micro.Post("/auth/login", authH.Login)

	micro.Post("/auth/logout", middleware.Protected(), authH.Logout)
	micro.Get("/toppings", middleware.Protected(), catalogH.ListToppings)
	micro.Post("/toppings", middleware.Protected(), catalogH.CreateTopping)
	micro.Post("/cash/open", middleware.Protected(), cashH.OpenRegister)
	micro.Get("/cash/current", middleware.Protected(), cashH.GetCurrent)
	micro.Post("/cash/close", middleware.Protected(), cashH.CloseRegister)
	micro.Get("/cash/history", middleware.Protected(), cashH.GetCashReports)
	micro.Get("/customers/lookup", middleware.Protected(), customerH.LookupByWhatsapp)
	micro.Post("/customers", middleware.Protected(), customerH.Create)
	micro.Get("/customers/:id", middleware.Protected(), customerH.GetByID)
	micro.Post("/orders", middleware.Protected(), orderH.CreateOrder)
	micro.Patch("/orders/:id/cancel", middleware.Protected(), orderH.CancelOrder)
	micro.Post("/monthly/close", middleware.Protected(), reportH.CloseMonth)
	micro.Get("/monthly", middleware.Protected(), reportH.GetMonthlyClosing)
	micro.Get("/monthly/history", middleware.Protected(), reportH.GetMonthlyClosingReports)

	micro.All("*", func(c *fiber.Ctx) error {
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Path: %v does not exists on this server", path),
		})
	})
}
