package bootstrap

import (
	"github.com/better-monitoring/bscout/api/routes"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, deps Dependencies) {
	apiV1 := app.Group("/api/v1")
	routes.EntryRouter(apiV1, deps.EntryService)

}
