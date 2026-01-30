package presenter

import (
	"fmt"
	"time"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/gofiber/fiber/v3"
)

func NightscoutStatusResponse() *fiber.Map {
	now := time.Now().UTC()
	serverTime := now.Format(time.RFC3339Nano)
	serverEpoch := now.UnixNano() / 1e6

	resp := fiber.Map{
		"status":          "ok",
		"name":            common.NAME,
		"version":         fmt.Sprintf("%d.%d.%d", common.MAJOR, common.MINOR, common.PATCH),
		"serverTime":      serverTime,
		"serverTimeEpoch": serverEpoch,
		"apiEnabled":      true,
	}

	return &resp
}
