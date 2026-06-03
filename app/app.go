package app

import (
	"canerollss/adapters/output/datasource"
	"canerollss/routes"
	"canerollss/utils/logs"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func App() {
	logs.InfoLog.Println("The oven is preheating")
	datasource.Connect()

	app := fiber.New()

	app.Static("/", "../uploads")
	app.Static("/", "./uploads") // Replace "./public" with the directory where your image files are located
	loggerConfig := logger.Config{
		Output: os.Stdout, // add file to save output
	}

	app.Use("/",
		cors.New(cors.Config{
			AllowCredentials: true,
			AllowOrigins:     "http://localhost:4200",
			AllowMethods:     "POST, PUT, DELETE, GET, OPTIONS",
		}),

		logger.New(loggerConfig),
	)

	routes.MapUrls(app)

	logs.InfoLog.Println("Fresh out of the oven")

	log.Fatal(app.Listen(":8080"))
}
