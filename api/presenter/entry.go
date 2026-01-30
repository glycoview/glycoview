package presenter

import (
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/gofiber/fiber/v3"
)

type Entry struct {
	ID         string `json:"_id,omitempty"`
	Type       string `json:"type"`
	DateString string `json:"dateString"`
	Date       int64  `json:"date"`
	SGV        int    `json:"sgv"`
	Direction  string `json:"direction"`
	Noise      int    `json:"noise"`
	Filtered   int    `json:"filtered"`
	Unfiltered int    `json:"unfiltered"`
	RSSI       int    `json:"rssi"`
}

func EntrySuccessResponse(entries *[]entities.Entry) *[]Entry {
	var response []Entry
	for _, data := range *entries {
		entry := Entry{
			ID:         data.ID,
			Type:       data.Type,
			DateString: data.DateString,
			Date:       data.Date,
			SGV:        data.SGV,
			Direction:  data.Direction,
			Noise:      data.Noise,
			Filtered:   data.Filtered,
			Unfiltered: data.Unfiltered,
			RSSI:       data.RSSI,
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
