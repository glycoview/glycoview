package bootstrap

import (
	"github.com/better-monitoring/bscout/api/routes"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, deps Dependencies) {
	apiV3 := app.Group("/api/v3")
	// register collection routers
	routes.DevicestatusRouter(apiV3, deps.DeviceStatusService)
	routes.EntryRouter(apiV3, deps.EntryService)

}
