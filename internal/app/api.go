package app

import (
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, deps Dependencies) {
	// v1Group := app.Group("/api/v1")
	// entryHandler := NewEntryHandler(deps.EntryService)

	// v1Group.Post("/entries", entryHandler.CreateEntry)
	// v1Group.Get("/entries/:id", entryHandler.GetEntry)
}
