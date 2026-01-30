package v1

import (
	v1 "github.com/better-monitoring/bscout/api/handlers/v1"
	"github.com/gofiber/fiber/v3"
)

func StatusRouter(apiV1 fiber.Router) {
	apiV1.Get("/status", v1.GetStatus())
}
