package handlers

import (
	"net/http"

	"github.com/better-monitoring/bscout/api/presenter"
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/better-monitoring/bscout/pkg/entry"
	"github.com/gofiber/fiber/v3"
)

func AddEntries(service entry.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		var requestBody []entities.Entry
		err := c.Bind().Body(&requestBody)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(presenter.EntryErrorResponse(err))
		}

		if err := service.CreateEntries(requestBody); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.EntryErrorResponse(err))
		}
		return c.JSON(presenter.EntrySuccessResponse(&[]entities.Entry{}))
	}
}
