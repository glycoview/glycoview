package ui

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/better-monitoring/glycoview/internal/model"
)

func glucoseSamples(records []model.Record) []GlucosePoint {
	points := make([]GlucosePoint, 0, len(records))
	for _, record := range records {
		value, ok := glucoseValue(record.Data)
		if !ok {
			continue
		}
		date, ok := model.Int64Field(record.Data, "date")
		if !ok {
			continue
		}
		direction, _ := model.StringField(record.Data, "direction")
		points = append(points, GlucosePoint{At: date, Value: value, Direction: direction})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].At < points[j].At })
	return points
}

func glucoseValue(data map[string]any) (float64, bool) {
	for _, field := range []string{"sgv", "mbg", "glucose"} {
		if value := model.PathValue(data, field); value != nil {
			if parsed, ok := floatValue(value); ok {
				return parsed, true
			}
		}
	}
	return 0, false
}

func latestSample(points []GlucosePoint) GlucosePoint {
	if len(points) == 0 {
		return GlucosePoint{}
	}
	return points[len(points)-1]
}

func filterRecentSamples(points []GlucosePoint, since time.Time) []GlucosePoint {
	filtered := make([]GlucosePoint, 0, len(points))
	cutoff := since.UnixMilli()
	for _, point := range points {
		if point.At >= cutoff {
			filtered = append(filtered, point)
		}
	}
	return filtered
}

func glucoseMetricValue(point GlucosePoint) string {
	if point.At == 0 || point.Value == 0 {
		return "No data"
	}
	return fmt.Sprintf("%.0f mg/dL", point.Value)
}

func glucoseDetail(point GlucosePoint) string {
	if point.At == 0 {
		return "Waiting for glucose data"
	}
	detail := time.UnixMilli(point.At).UTC().Format("15:04 UTC")
	if point.Direction != "" {
		detail += " · " + point.Direction
	}
	return detail
}

func glucoseAccent(value float64) string {
	switch {
	case value == 0:
		return "slate"
	case value < 70:
		return "rose"
	case value <= 180:
		return "green"
	case value <= 250:
		return "amber"
	default:
		return "orange"
	}
}

func timeInRange(points []GlucosePoint) []TimeInRangeBand {
	if len(points) == 0 {
		return []TimeInRangeBand{
			{Label: "Severe low", Range: "<54", Accent: "rose"},
			{Label: "Low", Range: "54-70", Accent: "pink"},
			{Label: "Target", Range: "70-180", Accent: "blue"},
			{Label: "High", Range: "180-250", Accent: "amber"},
			{Label: "Very high", Range: ">250", Accent: "orange"},
		}
	}
	var below54, low, target, high, veryHigh int
	for _, point := range points {
		switch {
		case point.Value < 54:
			below54++
		case point.Value < 70:
			low++
		case point.Value <= 180:
			target++
		case point.Value <= 250:
			high++
		default:
			veryHigh++
		}
	}
	total := maxInt(1, len(points))
	return []TimeInRangeBand{
		makeRangeBand("Severe low", "<54", below54, total, "rose"),
		makeRangeBand("Low", "54-70", low, total, "pink"),
		makeRangeBand("Target", "70-180", target, total, "blue"),
		makeRangeBand("High", "180-250", high, total, "amber"),
		makeRangeBand("Very high", ">250", veryHigh, total, "orange"),
	}
}

func narrowTimeInRange(points []GlucosePoint) TimeInRangeBand {
	if len(points) == 0 {
		return TimeInRangeBand{Label: "Tight target", Range: "70-140", Accent: "cyan"}
	}
	count := 0
	for _, point := range points {
		if point.Value >= 70 && point.Value <= 140 {
			count++
		}
	}
	return makeRangeBand("Tight target", "70-140", count, len(points), "cyan")
}

func makeRangeBand(label, rng string, count, total int, accent string) TimeInRangeBand {
	return TimeInRangeBand{
		Label:   label,
		Range:   rng,
		Minutes: count * 5,
		Percent: round1(float64(count) * 100 / float64(maxInt(total, 1))),
		Accent:  accent,
	}
}

func averageGlucose(points []GlucosePoint) float64 {
	if len(points) == 0 {
		return 0
	}
	var total float64
	for _, point := range points {
		total += point.Value
	}
	return total / float64(len(points))
}

func glucoseCV(points []GlucosePoint) float64 {
	if len(points) == 0 {
		return 0
	}
	avg := averageGlucose(points)
	if avg == 0 {
		return 0
	}
	var variance float64
	for _, point := range points {
		diff := point.Value - avg
		variance += diff * diff
	}
	sd := math.Sqrt(variance / float64(len(points)))
	return sd * 100 / avg
}

func sensorUsage(points []GlucosePoint, days int) float64 {
	expected := days * 24 * 12
	if expected <= 0 {
		return 0
	}
	return math.Min(100, float64(len(points))*100/float64(expected))
}

func buildAGP(points []GlucosePoint) []TrendBucket {
	buckets := make([][]float64, 24)
	for _, point := range points {
		hour := time.UnixMilli(point.At).UTC().Hour()
		buckets[hour] = append(buckets[hour], point.Value)
	}
	out := make([]TrendBucket, 0, 24)
	for hour := 0; hour < 24; hour++ {
		values := buckets[hour]
		sort.Float64s(values)
		out = append(out, TrendBucket{
			Hour:   hour,
			P10:    percentile(values, 10),
			P25:    percentile(values, 25),
			P50:    percentile(values, 50),
			P75:    percentile(values, 75),
			P90:    percentile(values, 90),
			Points: len(values),
		})
	}
	return out
}

func buildDailySummaries(points []GlucosePoint, treatments []model.Record) []DailySummary {
	groupedGlucose := map[string][]GlucosePoint{}
	groupedTreatments := map[string][]model.Record{}
	for _, point := range points {
		key := time.UnixMilli(point.At).UTC().Format("2006-01-02")
		groupedGlucose[key] = append(groupedGlucose[key], point)
	}
	for _, treatment := range treatments {
		if at, ok := model.Int64Field(treatment.Data, "date"); ok {
			key := time.UnixMilli(at).UTC().Format("2006-01-02")
			groupedTreatments[key] = append(groupedTreatments[key], treatment)
		}
	}
	keys := make([]string, 0, len(groupedGlucose))
	for key := range groupedGlucose {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]DailySummary, 0, len(keys))
	for _, key := range keys {
		dayPoints := groupedGlucose[key]
		carbs, insulin := totalsFromTreatments(groupedTreatments[key])
		tir := timeInRange(dayPoints)
		target := 0.0
		if len(tir) >= 3 {
			target = tir[2].Percent
		}
		parsed, _ := time.Parse("2006-01-02", key)
		out = append(out, DailySummary{
			Day:        parsed.Format("Mon"),
			Date:       parsed.UnixMilli(),
			AvgGlucose: averageGlucose(dayPoints),
			Carbs:      carbs,
			Insulin:    insulin,
			TIR:        target,
		})
	}
	return out
}
