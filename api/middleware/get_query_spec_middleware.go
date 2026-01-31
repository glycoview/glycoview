package middleware

import (
	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/gofiber/fiber/v3"
)

func GetQuerySpecMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		identifier := c.Params("spec")
		modified_since := c.GetHeaders()["If-Modified-Since"]

		spec, err := common.ParseGetQueryArgs(identifier, modified_since)
		if err != nil {
			return c.Status(400).JSON(map[string]any{"status": 400, "error": err.Error()})
		}
		c.Locals("querySpec", spec)
		return c.Next()
	}
}
