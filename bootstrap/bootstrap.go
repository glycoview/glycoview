package bootstrap

import (
	"github.com/better-monitoring/bscout/pkg/config"
	"github.com/gofiber/fiber/v3"
)

func Bootstrap(config *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		// ErrorHandler: middleware.ErrorHandler,
	})

	deps := buildDependencies(config)
	RegisterRoutes(app, deps)

	return app
}
