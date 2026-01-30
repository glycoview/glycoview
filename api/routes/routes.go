package routes

import (
	"github.com/better-monitoring/bscout/pkg/entry"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, entryService entry.IService) {
	apiV3 := app.Group("/api/v3")
	EntryRouter(apiV3, entryService)
}
