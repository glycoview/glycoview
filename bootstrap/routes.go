package bootstrap

import (
	"github.com/better-monitoring/bscout/api/routes"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, deps Dependencies) {
	apiV3 := app.Group("/api/v3")
	// register collection routers
	routes.EntryRouter(apiV3, deps.EntryService)
	routes.DevicestatusRouter(apiV3, deps.DeviceStatusService)
	routes.FoodRouter(apiV3, deps.FoodService)
	routes.ProfileRouter(apiV3, deps.ProfileService)
	routes.SettingsRouter(apiV3, deps.SettingsService)
	routes.TreatmentRouter(apiV3, deps.TreatmentService)

}
