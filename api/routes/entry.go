package routes

import (
	"github.com/better-monitoring/bscout/api/handlers"
	"github.com/better-monitoring/bscout/api/middleware"
	"github.com/better-monitoring/bscout/pkg/entry"
	"github.com/gofiber/fiber/v3"
)

func EntryRouter(apiV1 fiber.Router, service entry.IService) {
	apiV1.Get("/entries", middleware.QuerySpecMiddleware(), handlers.GetEntries(service))
	apiV1.Get("/entries/:spec", middleware.QuerySpecMiddleware(), handlers.GetEntriesWithIdOrTypeFiler(service))
	apiV1.Post("/entries", handlers.AddEntries(service))
	apiV1.Delete("/entries", middleware.QuerySpecMiddleware(), handlers.RemoveEntries(service))
}
