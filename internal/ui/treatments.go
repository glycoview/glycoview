package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/glycoview/glycoview/internal/model"
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

// splitTreatmentEvents returns typed slices for the different treatment kinds
// the clinical UI cares about. Legacy carbs/insulin slices are kept so older
// callers keep working; new callers should prefer the typed slices.
type treatmentSplit struct {
	Carbs      []EventPoint
	Insulin    []EventPoint
	Boluses    []EventPoint
	SMBs       []EventPoint
	TempBasals []EventPoint
	SMBGs      []EventPoint
}

func splitTreatmentEvents(records []model.Record) ([]EventPoint, []EventPoint) {
	split := splitTreatments(records)
	return split.Carbs, split.Insulin
}

func splitTreatments(records []model.Record) treatmentSplit {
	split := treatmentSplit{
		Carbs:      make([]EventPoint, 0),
		Insulin:    make([]EventPoint, 0),
		Boluses:    make([]EventPoint, 0),
		SMBs:       make([]EventPoint, 0),
		TempBasals: make([]EventPoint, 0),
		SMBGs:      make([]EventPoint, 0),
	}
	for _, record := range records {
		at, ok := model.Int64Field(record.Data, "date")
		if !ok {
			continue
		}
		eventType, _ := model.StringField(record.Data, "eventType")
		lowerType := strings.ToLower(eventType)

		carbValue, hasCarbs := floatValue(model.PathValue(record.Data, "carbs"))
		if hasCarbs && carbValue > 0 {
			split.Carbs = append(split.Carbs, EventPoint{At: at, Label: "Carbs", Kind: "carbs", Value: carbValue, Subtitle: eventType})
		}

		insulinValue, hasInsulin := floatValue(model.PathValue(record.Data, "insulin"))
		if hasInsulin && insulinValue > 0 {
			// Preserve the legacy generic "insulin" stream (everything injected).
			split.Insulin = append(split.Insulin, EventPoint{At: at, Label: "Bolus", Kind: "insulin", Value: insulinValue, Subtitle: eventType})
			// Classify the new typed streams.
			switch {
			case strings.Contains(lowerType, "smb"):
				split.SMBs = append(split.SMBs, EventPoint{At: at, Label: "SMB", Kind: "smb", Value: insulinValue, Subtitle: eventType})
			case strings.Contains(lowerType, "bolus"):
				split.Boluses = append(split.Boluses, EventPoint{At: at, Label: "Bolus", Kind: "bolus", Value: insulinValue, Subtitle: eventType})
			}
		}

		if strings.Contains(lowerType, "temp basal") || strings.Contains(lowerType, "temp-basal") || strings.Contains(lowerType, "temp_basal") {
			rate := 0.0
			if abs, ok := floatValue(model.PathValue(record.Data, "absolute")); ok {
				rate = abs
			} else if r, ok := floatValue(model.PathValue(record.Data, "rate")); ok {
				rate = r
			}
			duration := 0
			if d, ok := floatValue(model.PathValue(record.Data, "duration")); ok {
				duration = int(d)
			}
			split.TempBasals = append(split.TempBasals, EventPoint{
				At:       at,
				Label:    "Temp basal",
				Kind:     "temp-basal",
				Value:    rate,
				Duration: duration,
				Subtitle: eventType,
			})
		}

		if strings.Contains(lowerType, "bg check") || strings.Contains(lowerType, "glucose check") || strings.Contains(lowerType, "meter") {
			if g, ok := floatValue(model.PathValue(record.Data, "glucose")); ok && g > 0 {
				split.SMBGs = append(split.SMBGs, EventPoint{At: at, Label: "BG check", Kind: "smbg", Value: g, Subtitle: eventType})
			}
		}
	}
	return split
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
