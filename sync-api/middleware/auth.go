package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware checks if the X-API-Secret header matches the expected API_SECRET environment variable.
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := os.Getenv("API_SECRET")
		if secret == "" {
			// If no secret is configured, deny access by default or log a warning
			// In this case we return an error since auth is required.
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "API_SECRET is not configured on the server",
			})
		}

		clientSecret := c.Get("X-API-Secret")
		if clientSecret != secret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Invalid or missing X-API-Secret header",
			})
		}

		return c.Next()
	}
}
