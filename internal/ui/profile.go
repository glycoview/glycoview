package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/glycoview/glycoview/internal/model"
)

func sortFloats(values []float64) { sort.Float64s(values) }

func patientName(profile model.Record) string {
	for _, field := range []string{"patient.name", "name", "patient.fullName"} {
		if value := model.PathValue(profile.ToMap(true), field); value != nil {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
				return text
			}
		}
	}
	if profile.Identifier() != "" {
		return strings.ToUpper(profile.Identifier())
	}
	return "GlycoView Patient"
}

func buildBasalProfile(profile model.Record, day time.Time, statuses []model.Record) []EventPoint {
	points := make([]EventPoint, 0)
	schedule := scheduleSlice(profile, "basal")
	base := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	for _, item := range schedule {
		at := parseClock(base, item.Time)
		value, _ := floatValue(item.Value)
		points = append(points, EventPoint{At: at.UnixMilli(), Label: "Basal", Kind: "basal", Value: value, Subtitle: item.Time})
	}
	// Fallback: if no profile schedule, expose a single flat rate derived
	// from the median of the day's openaps-enacted temp basals. This gives the
	// chart something sensible to draw when Nightscout has no profile record.
	if len(points) == 0 && len(statuses) > 0 {
		if rate, ok := medianEnactedRate(statuses); ok {
			points = append(points, EventPoint{At: base.UnixMilli(), Label: "Basal", Kind: "basal", Value: rate, Subtitle: "derived"})
		}
	}
	return points
}

func medianEnactedRate(statuses []model.Record) (float64, bool) {
	rates := make([]float64, 0, len(statuses))
	for _, record := range statuses {
		if rate, ok := floatValue(model.PathValue(record.Data, "openaps.enacted.rate")); ok && rate >= 0 {
			rates = append(rates, rate)
		}
	}
	if len(rates) == 0 {
		return 0, false
	}
	return percentile(sortedFloats(rates), 50), true
}

func sortedFloats(values []float64) []float64 {
	out := append([]float64(nil), values...)
	sortFloats(out)
	return out
}

func profileHeadline(profile model.Record) string {
	defaultProfile, _ := model.StringField(profile.Data, "defaultProfile")
	startDate, _ := model.StringField(profile.Data, "startDate")
	if defaultProfile == "" {
		defaultProfile = "Default profile"
	}
	if startDate == "" {
		return defaultProfile
	}
	return defaultProfile + " · active from " + startDate
}

func buildProfileMetrics(profile model.Record) []Metric {
	defaultProfile, _ := model.StringField(profile.Data, "defaultProfile")
	startDate, _ := model.StringField(profile.Data, "startDate")
	storeMap, _ := model.PathValue(profile.Data, "store").(map[string]any)
	return []Metric{
		{ID: "profile", Label: "Default profile", Value: firstNonEmpty(defaultProfile, "n/a"), Accent: "blue"},
		{ID: "start", Label: "Start date", Value: firstNonEmpty(startDate, "n/a"), Accent: "violet"},
		{ID: "variants", Label: "Stored profiles", Value: strconv.Itoa(len(storeMap)), Accent: "slate"},
		{ID: "updated", Label: "Last server update", Value: time.UnixMilli(profile.SrvModified).UTC().Format("02 Jan 2006 15:04"), Accent: "green"},
	}
}

func profileSchedule(profile model.Record, scheduleType string) []SchedulePoint {
	return scheduleSlice(profile, scheduleType)
}

func scheduleSlice(profile model.Record, scheduleType string) []SchedulePoint {
	active := activeProfileMap(profile)
	raw, _ := active[scheduleType].([]any)
	points := make([]SchedulePoint, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		points = append(points, SchedulePoint{
			Time:  strings.TrimSpace(fmt.Sprint(entry["time"])),
			Value: strings.TrimSpace(fmt.Sprint(entry["value"])),
		})
	}
	return points
}

func profileTargets(profile model.Record) []SchedulePoint {
	active := activeProfileMap(profile)
	raw, _ := active["target_low"].([]any)
	points := make([]SchedulePoint, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		timeLabel := strings.TrimSpace(fmt.Sprint(entry["time"]))
		low := strings.TrimSpace(fmt.Sprint(entry["value"]))
		high := low
		if highs, ok := active["target_high"].([]any); ok {
			for _, hi := range highs {
				if hiMap, ok := hi.(map[string]any); ok && strings.TrimSpace(fmt.Sprint(hiMap["time"])) == timeLabel {
					high = strings.TrimSpace(fmt.Sprint(hiMap["value"]))
					break
				}
			}
		}
		points = append(points, SchedulePoint{Time: timeLabel, Value: low + " - " + high})
	}
	return points
}

func activeProfileMap(profile model.Record) map[string]any {
	storeMap, _ := model.PathValue(profile.Data, "store").(map[string]any)
	defaultProfile, _ := model.StringField(profile.Data, "defaultProfile")
	if storeMap != nil {
		if current, ok := storeMap[defaultProfile].(map[string]any); ok {
			return current
		}
		for _, value := range storeMap {
			if current, ok := value.(map[string]any); ok {
				return current
			}
		}
	}
	return map[string]any{}
}
