package app

import (
	"canerollss/adapters/input/http/handlers"
	"canerollss/adapters/input/http/routes"
	"canerollss/adapters/output/datasource"
	"canerollss/adapters/output/delivery"
	"canerollss/adapters/output/payment"
	"canerollss/adapters/output/repository"
	"canerollss/adapters/output/whatsapp"
	"canerollss/core/usecase"
	"canerollss/ports/output"
	"canerollss/utils/logs"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupApp() *fiber.App {
	datasource.Connect()
	db := datasource.GetDB()

	userRepo := repository.NewUserRepository(db)
	cashRepo := repository.NewCashRegisterRepository(db)
	toppingRepo := repository.NewToppingRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reportRepo := repository.NewMonthlyClosingRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	customerAddressRepo := repository.NewCustomerAddressRepository(db)

	authUC := usecase.NewAuthUseCase(userRepo)
	cashUC := usecase.NewCashUseCase(cashRepo, orderRepo)
	catalogUC := usecase.NewCatalogUseCase(toppingRepo)
	orderUC := usecase.NewOrderUseCase(orderRepo, cashRepo, customerRepo, toppingRepo)
	reportUC := usecase.NewReportUseCase(reportRepo, orderRepo)
	customerUC := usecase.NewCustomerUseCase(customerRepo)
	addressUC := usecase.NewAddressUseCase(customerAddressRepo)

	// External logistics & payment providers (Mock by default or configured)
	useMock := os.Getenv("USE_MOCK_PROVIDERS")
	var deliveryPort output.DeliveryProvider
	var paymentPort output.PaymentProvider

	if strings.EqualFold(useMock, "false") && os.Getenv("UBER_DIRECT_CUSTOMER_ID") != "" {
		deliveryPort = delivery.NewUberDirectProvider()
	} else {
		deliveryPort = delivery.NewMockDeliveryProvider()
	}

	if strings.EqualFold(useMock, "false") && os.Getenv("MP_ACCESS_TOKEN") != "" {
		paymentPort = payment.NewMercadoPagoProvider()
	} else {
		paymentPort = payment.NewMockPaymentProvider()
	}

	waClient := whatsapp.NewWhatsAppClient()
	reconUC := usecase.NewPaymentReconciliationUseCase(orderRepo, cashRepo, paymentPort)
	agentUC := usecase.NewWhatsAppAgentUseCase(orderUC, addressUC, customerRepo, toppingRepo, cashRepo, deliveryPort, paymentPort, reconUC)

	authHandler := handlers.NewAuthHandler(authUC)
	cashHandler := handlers.NewCashHandler(cashUC)
	catalogHandler := handlers.NewCatalogHandler(catalogUC)
	orderHandler := handlers.NewOrderHandler(orderUC)
	reportHandler := handlers.NewReportHandler(reportUC)
	customerHandler := handlers.NewCustomerHandler(customerUC)
	waHandler := handlers.NewWhatsAppHandler(agentUC, waClient)
	mpWebhookSecret := os.Getenv("MP_WEBHOOK_SECRET")
	mpHandler := handlers.NewMercadoPagoHandler(orderUC, paymentPort, reconUC, waClient, mpWebhookSecret)

	app := fiber.New()

	app.Static("/", "../uploads")
	app.Static("/", "./uploads")

	loggerConfig := logger.Config{
		Output: os.Stdout,
	}

	app.Use("/",
		logger.New(loggerConfig),
	)

	routes.MapUrls(app, authHandler, cashHandler, catalogHandler, orderHandler, reportHandler, customerHandler)
	routes.MapWebhookRoutes(app, waHandler, mpHandler)

	return app
}

func Start() {
	logs.InfoLog.Println("The oven is preheating")

	app := SetupApp()

	logs.InfoLog.Println("Fresh out of the oven")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
