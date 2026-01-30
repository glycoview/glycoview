package app

import (
	"github.com/better-monitoring/bscout/internal/config"
	"github.com/better-monitoring/bscout/internal/middleware"
	"github.com/gofiber/fiber/v3"
)

func NewServer(config *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	middleware.RegisterGlobal(app, config)

	deps := BuildDependencies(config)
	RegisterRoutes(app, deps)

	return app
}
