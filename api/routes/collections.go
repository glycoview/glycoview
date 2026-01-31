package routes

import (
	"github.com/better-monitoring/bscout/api/handlers"
	"github.com/better-monitoring/bscout/api/middleware"
	"github.com/better-monitoring/bscout/pkg/devicestatus"
	"github.com/better-monitoring/bscout/pkg/entry"
	"github.com/gofiber/fiber/v3"
)

func DevicestatusRouter(api fiber.Router, svc devicestatus.IService) {
	api.Get("/devicestatus", middleware.SearchQuerySpecMiddleware(), handlers.SearchDeviceStatus(svc))
	api.Get("/devicestatus/:identifier", middleware.GetQuerySpecMiddleware(), handlers.GetOneDeviceStatus(svc))
	api.Post("/devicestatus", handlers.AddDeviceStatus(svc))
	api.Delete("/devicestatus", middleware.SearchQuerySpecMiddleware(), handlers.RemoveDeviceStatus(svc))
}

func EntryRouter(api fiber.Router, svc entry.IService) {
	api.Get("/entries", middleware.SearchQuerySpecMiddleware(), handlers.SearchEntries(svc))
	api.Get("/entries/:identifier", middleware.GetQuerySpecMiddleware(), handlers.GetOneEntry(svc))
	api.Post("/entries", handlers.AddEntry(svc))
	api.Delete("/entries", middleware.SearchQuerySpecMiddleware(), handlers.RemoveEntry(svc))
}
