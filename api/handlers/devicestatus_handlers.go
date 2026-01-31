package handlers

import (
	"net/http"

	"github.com/better-monitoring/bscout/api/presenter"
	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/devicestatus"
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/gofiber/fiber/v3"
)

// Device Status
func SearchDeviceStatus(svc devicestatus.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		items, err := svc.Search(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(http.StatusInternalServerError, err))
		}
		res := presenter.DeviceStatusSuccessResponse(items)
		c.Status(http.StatusOK)
		return c.JSON(presenter.SearchListResponse(res))
	}
}

func GetOneDeviceStatus(svc devicestatus.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		item, err := svc.GetOne(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(http.StatusInternalServerError, err))
		}
		res := presenter.DeviceStatusFromEntity(item)
		c.Status(http.StatusOK)
		return c.JSON(presenter.SearchObjectResponse(res))
	}
}

func AddDeviceStatus(svc devicestatus.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		var body []entities.DeviceStatus
		if err := c.Bind().Body(&body); err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(presenter.ErrorResponse(http.StatusBadRequest, err))
		}
		if err := svc.Create(body); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(http.StatusInternalServerError, err))
		}
		c.Status(http.StatusCreated)
		return c.JSON(presenter.CreateResponseCreated(""))
	}
}

func RemoveDeviceStatus(svc devicestatus.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		if err := svc.Remove(spec); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(http.StatusInternalServerError, err))
		}
		return c.JSON(&fiber.Map{"status": 200})
	}
}
