package presenter

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

func ErrorResponse(status int, err error) *fiber.Map {
	return &fiber.Map{"status": status, "error": err.Error()}
}

func SearchListResponse[T any](data []*T) *fiber.Map {
	return &fiber.Map{"status": 200, "result": data}
}

func SearchObjectResponse[T any](data *T) *fiber.Map {
	return &fiber.Map{"status": 200, "result": data}
}

// CreateResponseCreated returns inline_response_201
func CreateResponseCreated(identifier string) *fiber.Map {
	return &fiber.Map{"status": 201, "identifier": identifier, "lastModified": time.Now().UnixNano() / 1e6}
}

// OkResponse returns a 200 wrapper
func OkResponse() *fiber.Map { return &fiber.Map{"status": 200} }
