package middleware

import (
	"errors"
	"log"

	app_error "github.com/better-monitoring/bscout/internal/error"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

func ErrorHandler(c fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	response := ErrorResponse{
		Error: "internal server error",
	}

	var fe *fiber.Error
	if errors.As(err, &fe) {
		status = fe.Code
		response.Error = fe.Message

		return c.Status(status).JSON(response)
	}

	var appErr *app_error.AppError
	if errors.As(err, &appErr) {
		status = appErr.Status
		response.Error = appErr.Message
		response.Code = appErr.Code

		return c.Status(status).JSON(response)
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		status = fiber.StatusBadRequest
		response.Error = "validation failed"
		response.Details = ve.Error()

		return c.Status(status).JSON(response)
	}

	log.Printf("UNHANDLED ERROR: %v", err)

	return c.Status(status).JSON(response)
}
