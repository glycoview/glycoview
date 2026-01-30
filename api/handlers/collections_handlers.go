package handlers

import (
	"net/http"

	"github.com/better-monitoring/bscout/api/presenter"
	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/devicestatus"
	"github.com/better-monitoring/bscout/pkg/food"
	"github.com/better-monitoring/bscout/pkg/profile"
	"github.com/better-monitoring/bscout/pkg/settings"
	"github.com/better-monitoring/bscout/pkg/treatment"
	"github.com/gofiber/fiber/v3"
)

func GetDeviceStatus(svc devicestatus.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		_, err := svc.Search(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.SearchResponse())
	}
}

func AddDeviceStatus(svc devicestatus.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		// placeholder: accept any body and return created
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
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.OkResponse())
	}
}

// Food handlers
func GetFood(svc food.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		_, err := svc.Search(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.SearchResponse())
	}
}

func AddFood(svc food.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Status(http.StatusCreated)
		return c.JSON(presenter.CreateResponseCreated(""))
	}
}

func RemoveFood(svc food.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		if err := svc.Remove(spec); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.OkResponse())
	}
}

// Profile handlers
func GetProfile(svc profile.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		_, err := svc.Search(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.SearchResponse())
	}
}

func AddProfile(svc profile.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Status(http.StatusCreated)
		return c.JSON(presenter.CreateResponseCreated(""))
	}
}

func RemoveProfile(svc profile.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		if err := svc.Remove(spec); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.OkResponse())
	}
}

// Settings handlers
func GetSettings(svc settings.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		_, err := svc.Search(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.SearchResponse())
	}
}

func AddSettings(svc settings.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Status(http.StatusCreated)
		return c.JSON(presenter.CreateResponseCreated(""))
	}
}

func RemoveSettings(svc settings.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		if err := svc.Remove(spec); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.OkResponse())
	}
}

// Treatments handlers
func GetTreatments(svc treatment.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		_, err := svc.Search(spec)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.SearchResponse())
	}
}

func AddTreatments(svc treatment.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Status(http.StatusCreated)
		return c.JSON(presenter.CreateResponseCreated(""))
	}
}

func RemoveTreatments(svc treatment.IService) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Locals("querySpec")
		spec, _ := raw.(*common.QuerySpec)
		if spec == nil {
			spec = &common.QuerySpec{}
		}
		if err := svc.Remove(spec); err != nil {
			c.Status(http.StatusInternalServerError)
			return c.JSON(presenter.ErrorResponse(err))
		}
		return c.JSON(presenter.OkResponse())
	}
}
