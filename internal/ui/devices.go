package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/better-monitoring/glycoview/internal/model"
)

func latestStatusActivity(statuses []model.Record, limit int) []ActivityItem {
	items := make([]ActivityItem, 0, len(statuses))
	for _, record := range statuses {
		at, ok := model.Int64Field(record.Data, "date")
		if !ok {
			continue
		}
		device, _ := model.StringField(record.Data, "device")
		items = append(items, ActivityItem{
			At:     at,
			Title:  firstNonEmpty(device, "Device status"),
			Detail: deviceSummary(record.Data),
			Kind:   "device",
			Accent: "blue",
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].At > items[j].At })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func buildDeviceCards(statuses []model.Record) []DeviceCard {
	latestByDevice := map[string]model.Record{}
	for _, record := range statuses {
		device, _ := model.StringField(record.Data, "device")
		if device == "" {
			device = "Unknown device"
		}
		current := latestByDevice[device]
		if current.Identifier() == "" || record.SrvModified > current.SrvModified {
			latestByDevice[device] = record
		}
	}
	keys := make([]string, 0, len(latestByDevice))
	for key := range latestByDevice {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	cards := make([]DeviceCard, 0, len(keys))
	for _, key := range keys {
		record := latestByDevice[key]
		date, _ := model.Int64Field(record.Data, "date")
		cards = append(cards, DeviceCard{
			Name:       key,
			Kind:       deviceKind(key),
			Status:     deviceSummary(record.Data),
			LastSeen:   date,
			Battery:    firstNumberString(record.Data, "uploaderBattery", "pump.battery.percent", "battery"),
			Reservoir:  firstNumberString(record.Data, "pump.reservoir", "reservoir"),
			Badge:      loopBadge(record.Data),
			Connection: connectionState(record.Data),
			Details: []Metric{
				{Label: "Loop", Value: loopBadge(record.Data), Accent: "violet"},
				{Label: "Uploader battery", Value: firstNonEmpty(firstNumberString(record.Data, "uploaderBattery", "battery"), "n/a"), Accent: "green"},
			},
		})
	}
	return cards
}

func deviceKind(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "dexcom"), strings.Contains(lower, "cgm"):
		return "CGM"
	case strings.Contains(lower, "pump"), strings.Contains(lower, "roche"), strings.Contains(lower, "medtronic"):
		return "Pump"
	case strings.Contains(lower, "loop"), strings.Contains(lower, "xdrip"), strings.Contains(lower, "rig"):
		return "Loop bridge"
	default:
		return "Integration"
	}
}

func deviceSummary(data map[string]any) string {
	parts := make([]string, 0, 3)
	if loop := loopBadge(data); loop != "" {
		parts = append(parts, loop)
	}
	if reservoir := firstNumberString(data, "pump.reservoir", "reservoir"); reservoir != "" {
		parts = append(parts, "Reservoir "+reservoir)
	}
	if battery := firstNumberString(data, "uploaderBattery", "pump.battery.percent", "battery"); battery != "" {
		parts = append(parts, "Battery "+battery)
	}
	if len(parts) == 0 {
		return "Latest Nightscout device feed is available"
	}
	return strings.Join(parts, " · ")
}

func connectionState(data map[string]any) string {
	for _, field := range []string{"pump.connection", "connection", "status"} {
		if value := model.PathValue(data, field); value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	return "Connected"
}

func loopBadge(data map[string]any) string {
	for _, field := range []string{"loop.mode", "openaps.mode", "loop.status"} {
		if value := model.PathValue(data, field); value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	for _, field := range []string{"openaps.suggested", "openaps.enacted", "loop"} {
		if value := model.PathValue(data, field); value != nil {
			return "Loop active"
		}
	}
	return "Monitoring"
}

func countFreshDevices(cards []DeviceCard, now time.Time) int {
	count := 0
	cutoff := now.Add(-24 * time.Hour).UnixMilli()
	for _, card := range cards {
		if card.LastSeen >= cutoff {
			count++
		}
	}
	return count
}
