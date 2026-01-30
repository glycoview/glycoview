package handlers

import (
	"github.com/better-monitoring/bscout/api/presenter"
	"github.com/gofiber/fiber/v3"
)

func GetStatus() fiber.Handler {
	return func(c fiber.Ctx) error {

		return c.JSON(presenter.NightscoutStatusResponse())
	}
}
