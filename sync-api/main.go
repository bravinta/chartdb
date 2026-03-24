package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New()

	app.Use(logger.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	log.Println("Starting sync-api server on port 3001...")
	if err := app.Listen(":3001"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
