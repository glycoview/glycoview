package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/better-monitoring/glycoview/internal/model"
)

func totalsFromTreatments(records []model.Record) (float64, float64) {
	var carbs, insulin float64
	for _, record := range records {
		if value, ok := floatValue(model.PathValue(record.Data, "carbs")); ok {
			carbs += value
		}
		if value, ok := floatValue(model.PathValue(record.Data, "insulin")); ok {
			insulin += value
		}
	}
	return carbs, insulin
}

func splitTreatmentEvents(records []model.Record) ([]EventPoint, []EventPoint) {
	carbs := make([]EventPoint, 0)
	insulin := make([]EventPoint, 0)
	for _, record := range records {
		at, ok := model.Int64Field(record.Data, "date")
		if !ok {
			continue
		}
		eventType, _ := model.StringField(record.Data, "eventType")
		if value, ok := floatValue(model.PathValue(record.Data, "carbs")); ok && value > 0 {
			carbs = append(carbs, EventPoint{At: at, Label: "Carbs", Kind: "carbs", Value: value, Subtitle: eventType})
		}
		if value, ok := floatValue(model.PathValue(record.Data, "insulin")); ok && value > 0 {
			insulin = append(insulin, EventPoint{At: at, Label: "Bolus", Kind: "insulin", Value: value, Subtitle: eventType})
		}
	}
	return carbs, insulin
}

func buildRecentActivity(treatments, statuses []model.Record, limit int) []ActivityItem {
	items := make([]ActivityItem, 0, len(treatments)+len(statuses))
	for _, record := range treatments {
		at, ok := model.Int64Field(record.Data, "date")
		if !ok {
			continue
		}
		eventType, _ := model.StringField(record.Data, "eventType")
		carbs, _ := floatValue(model.PathValue(record.Data, "carbs"))
		insulin, _ := floatValue(model.PathValue(record.Data, "insulin"))
		detail := strings.TrimSpace(strings.Join([]string{
			nonEmpty(fmt.Sprintf("%.0fg carbs", carbs), carbs > 0),
			nonEmpty(fmt.Sprintf("%.1fU insulin", insulin), insulin > 0),
		}, " · "))
		if detail == "" {
			detail = "Treatment recorded"
		}
		items = append(items, ActivityItem{
			At: at, Title: firstNonEmpty(eventType, "Treatment"), Detail: detail, Kind: "treatment", Accent: "amber",
		})
	}
	for _, record := range statuses {
		at, ok := model.Int64Field(record.Data, "date")
		if !ok {
			continue
		}
		device, _ := model.StringField(record.Data, "device")
		items = append(items, ActivityItem{
			At: at, Title: firstNonEmpty(device, "Device status"), Detail: deviceSummary(record.Data), Kind: "device", Accent: "blue",
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].At > items[j].At })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
