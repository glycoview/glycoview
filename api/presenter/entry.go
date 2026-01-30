package presenter

import (
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/gofiber/fiber/v3"
)

type Entry struct {
	DocumentBase
	Type       string `json:"type"`
	DateString string `json:"dateString,omitempty"`
	SGV        *int   `json:"sgv,omitempty"`
	Direction  string `json:"direction,omitempty"`
	Noise      *int   `json:"noise,omitempty"`
	Filtered   *int   `json:"filtered,omitempty"`
	Unfiltered *int   `json:"unfiltered,omitempty"`
	RSSI       *int   `json:"rssi,omitempty"`
	Units      string `json:"units,omitempty"`
}

func EntrySuccessResponse(entries *[]entities.Entry) *[]Entry {
	var response []Entry
	for _, data := range *entries {
		base := DocumentBaseFromEntity(data.DocumentBase)
		entry := Entry{
			DocumentBase: base,
			Type:         data.Type,
			DateString:   data.DateString,
			SGV:          data.SGV,
			Direction:    data.Direction,
			Noise:        data.Noise,
			Filtered:     data.Filtered,
			Unfiltered:   data.Unfiltered,
			RSSI:         data.RSSI,
			Units:        data.Units,
		}
		response = append(response, entry)
	}
	return &response
}

func EntryErrorResponse(err error) *fiber.Map {
	return &fiber.Map{
		"code":    500,
		"message": err.Error(),
		"fields":  map[string]interface{}{},
	}
}
