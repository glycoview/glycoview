package presenter

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ErrorResponse returns a standard error wrapper
func ErrorResponse(err error) *fiber.Map {
	return &fiber.Map{"status": http.StatusInternalServerError, "error": err.Error()}
}

// SearchResponse returns an empty 200 search result
func SearchResponse() *fiber.Map {
	return &fiber.Map{"status": 200, "result": []interface{}{}}
}

// CreateResponseCreated returns inline_response_201
func CreateResponseCreated(identifier string) *fiber.Map {
	return &fiber.Map{"status": 201, "identifier": identifier, "lastModified": time.Now().UnixNano() / 1e6}
}

// OkResponse returns a 200 wrapper
func OkResponse() *fiber.Map { return &fiber.Map{"status": 200} }
