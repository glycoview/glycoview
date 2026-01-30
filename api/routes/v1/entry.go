package v1

import (
	v1 "github.com/better-monitoring/bscout/api/handlers/v1"
	"github.com/better-monitoring/bscout/pkg/entry"
	"github.com/gofiber/fiber/v3"
)

func EntryRouter(apiV1 fiber.Router, service entry.IService) {
	apiV1.Get("/entries", v1.GetEntries(service))
	apiV1.Get("/entries/:spec", v1.GetEntriesWithIdOrTypeFiler(service))
	apiV1.Post("/entries", v1.AddEntries(service))
	apiV1.Delete("/entries", v1.RemoveEntries(service))
}
