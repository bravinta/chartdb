package main

import (
	"log"
	"os"

	"sync-api/handlers"
	"sync-api/middleware"
	"sync-api/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	app := fiber.New()

	app.Use(logger.New())

	// Connect to database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=chartdb_sync port=5432 sslmode=disable TimeZone=UTC"
	}

	log.Println("Connecting to database...")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate
	log.Println("Running AutoMigrate...")
	err = db.AutoMigrate(
		&models.Diagram{},
		&models.DBTable{},
		&models.DBRelationship{},
		&models.DBDependency{},
	)
	if err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Setup routes
	syncHandler := handlers.NewSyncHandler(db)

	api := app.Group("/api")
	syncGroup := api.Group("/sync", middleware.AuthMiddleware())

	syncGroup.Get("/pull/:diagramId", syncHandler.Pull)
	syncGroup.Post("/push", syncHandler.Push)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("Starting sync-api server on port %s...\n", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
