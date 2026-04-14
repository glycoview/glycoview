package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/glycoview/glycoview/internal/model"
)

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	position := (p / 100) * float64(len(values)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return values[lower]
	}
	weight := position - float64(lower)
	return values[lower] + (values[upper]-values[lower])*weight
}

func parseClock(base time.Time, clock string) time.Time {
	parts := strings.Split(clock, ":")
	if len(parts) < 2 {
		return base
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, time.UTC)
}

func firstNumberString(data map[string]any, fields ...string) string {
	for _, field := range fields {
		if value := model.PathValue(data, field); value != nil {
			if parsed, ok := floatValue(value); ok {
				if parsed == math.Trunc(parsed) {
					return fmt.Sprintf("%.0f", parsed)
				}
				return fmt.Sprintf("%.1f", parsed)
			}
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func floatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nonEmpty(value string, include bool) string {
	if include {
		return value
	}
	return ""
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
