package middleware

import (
	"github.com/better-monitoring/bscout/internal/config"
	"github.com/gofiber/fiber/v3"
)

func RegisterGlobal(app *fiber.App, cfg *config.Config) {
	// app.Use(recover.New())
	// app.Use(logger.New())
	// app.Use(middleware.RequestID())
}
