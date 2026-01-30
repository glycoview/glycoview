package bootstrap

import (
	v1 "github.com/better-monitoring/bscout/api/routes/v1"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, deps Dependencies) {
	apiV1 := app.Group("/api/v1")
	v1.EntryRouter(apiV1, deps.EntryService)
	v1.StatusRouter(apiV1)
}
