package app

import (
	"canerollss/adapters/input/http/handlers"
	"canerollss/adapters/input/http/routes"
	"canerollss/adapters/output/datasource"
	"canerollss/adapters/output/repository"
	"canerollss/core/usecase"
	"canerollss/utils/logs"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func Start() {
	logs.InfoLog.Println("The oven is preheating")

	datasource.Connect()
	db := datasource.GetDB()

	userRepo := repository.NewUserRepository(db)
	cashRepo := repository.NewCashRegisterRepository(db)
	toppingRepo := repository.NewToppingRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reportRepo := repository.NewMonthlyClosingRepository(db)
	customerRepo := repository.NewCustomerRepository(db)

	authUC := usecase.NewAuthUseCase(userRepo)
	cashUC := usecase.NewCashUseCase(cashRepo, orderRepo)
	catalogUC := usecase.NewCatalogUseCase(toppingRepo)
	orderUC := usecase.NewOrderUseCase(orderRepo, cashRepo, customerRepo, toppingRepo)
	reportUC := usecase.NewReportUseCase(reportRepo, orderRepo)

	authHandler := handlers.NewAuthHandler(authUC)
	cashHandler := handlers.NewCashHandler(cashUC)
	catalogHandler := handlers.NewCatalogHandler(catalogUC)
	orderHandler := handlers.NewOrderHandler(orderUC)
	reportHandler := handlers.NewReportHandler(reportUC)

	app := fiber.New()

	app.Static("/", "../uploads")
	app.Static("/", "./uploads")

	loggerConfig := logger.Config{
		Output: os.Stdout,
	}

	app.Use("/",
		cors.New(cors.Config{
			AllowCredentials: true,
			AllowOrigins:     "http://localhost:4200",
			AllowMethods:     "POST, PUT, DELETE, GET, OPTIONS",
		}),
		logger.New(loggerConfig),
	)

	routes.MapUrls(app, authHandler, cashHandler, catalogHandler, orderHandler, reportHandler)

	logs.InfoLog.Println("Fresh out of the oven")

	log.Fatal(app.Listen(":8080"))
}
