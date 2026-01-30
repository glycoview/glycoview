package routes

import (
	"github.com/better-monitoring/bscout/api/handlers"
	"github.com/better-monitoring/bscout/api/middleware"
	"github.com/better-monitoring/bscout/pkg/devicestatus"
	"github.com/better-monitoring/bscout/pkg/food"
	"github.com/better-monitoring/bscout/pkg/profile"
	"github.com/better-monitoring/bscout/pkg/settings"
	"github.com/better-monitoring/bscout/pkg/treatment"
	"github.com/gofiber/fiber/v3"
)

func DevicestatusRouter(api fiber.Router, svc devicestatus.IService) {
	api.Get("/devicestatus", middleware.QuerySpecMiddleware(), handlers.GetDeviceStatus(svc))
	api.Post("/devicestatus", handlers.AddDeviceStatus(svc))
	api.Delete("/devicestatus", middleware.QuerySpecMiddleware(), handlers.RemoveDeviceStatus(svc))
}

func FoodRouter(api fiber.Router, svc food.IService) {
	api.Get("/food", middleware.QuerySpecMiddleware(), handlers.GetFood(svc))
	api.Post("/food", handlers.AddFood(svc))
	api.Delete("/food", middleware.QuerySpecMiddleware(), handlers.RemoveFood(svc))
}

func ProfileRouter(api fiber.Router, svc profile.IService) {
	api.Get("/profile", middleware.QuerySpecMiddleware(), handlers.GetProfile(svc))
	api.Post("/profile", handlers.AddProfile(svc))
	api.Delete("/profile", middleware.QuerySpecMiddleware(), handlers.RemoveProfile(svc))
}

func SettingsRouter(api fiber.Router, svc settings.IService) {
	api.Get("/settings", middleware.QuerySpecMiddleware(), handlers.GetSettings(svc))
	api.Post("/settings", handlers.AddSettings(svc))
	api.Delete("/settings", middleware.QuerySpecMiddleware(), handlers.RemoveSettings(svc))
}

func TreatmentRouter(api fiber.Router, svc treatment.IService) {
	api.Get("/treatments", middleware.QuerySpecMiddleware(), handlers.GetTreatments(svc))
	api.Post("/treatments", handlers.AddTreatments(svc))
	api.Delete("/treatments", middleware.QuerySpecMiddleware(), handlers.RemoveTreatments(svc))
}
