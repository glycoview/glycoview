package bootstrap

import (
	"github.com/better-monitoring/bscout/api/routes"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, deps Dependencies) {
	apiV3 := app.Group("/api/v3")
	routes.EntryRouter(apiV3, deps.EntryService)

}
