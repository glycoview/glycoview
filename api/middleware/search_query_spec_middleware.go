package middleware

import (
	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/gofiber/fiber/v3"
)

func SearchQuerySpecMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		spec, err := common.ParseSearchQueryArgs(c.Queries())
		if err != nil {
			return c.Status(400).JSON(map[string]any{"status": 400, "error": err.Error()})
		}
		c.Locals("querySpec", spec)
		return c.Next()
	}
}
