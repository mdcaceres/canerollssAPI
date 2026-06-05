package routes

import (
	"canerollss/adapters/input/http/handlers"
	"canerollss/adapters/input/http/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func MapUrls(app *fiber.App, authH *handlers.AuthHandler, cashH *handlers.CashHandler, catalogH *handlers.CatalogHandler, orderH *handlers.OrderHandler, reportH *handlers.ReportHandler) {
	micro := app.Group("/api")

	micro.Post("/auth/register", authH.Register)
	micro.Post("/auth/login", authH.Login)

	micro.Post("/auth/logout", middleware.Protected(), authH.Logout)
	micro.Get("/toppings", middleware.Protected(), catalogH.ListToppings)
	micro.Post("/toppings", middleware.Protected(), catalogH.CreateTopping)
	micro.Post("/cash/open", middleware.Protected(), cashH.OpenRegister)
	micro.Post("/cash/close", middleware.Protected(), cashH.CloseRegister)
	micro.Post("/orders", middleware.Protected(), orderH.CreateOrder)
	micro.Patch("/orders/:id/cancel", middleware.Protected(), orderH.CancelOrder)
	micro.Post("/reports/monthly/close", middleware.Protected(), reportH.CloseMonth)
	micro.Get("/reports/monthly", middleware.Protected(), reportH.GetMonthlyClosing)
	micro.Get("/reports/history", middleware.Protected(), reportH.GetMonthlyClosingReports)

	micro.All("*", func(c *fiber.Ctx) error {
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Path: %v does not exists on this server", path),
		})
	})
}
