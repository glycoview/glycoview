package goals

import (
	"math"
	"sort"
	"time"
)

// Sample is a minimal glucose reading. We keep this independent of
// model/ui packages so the evaluator is easy to unit-test.
type Sample struct {
	At    time.Time
	Value float64
}

// EvalInput bundles the raw data the aggregates need.
type EvalInput struct {
	Samples []Sample
	TZ      *time.Location
}

// ComputeAggregate evaluates a predicate's LHS against the samples in
// window. Returns the numeric value in the aggregate's natural unit:
//
//	 tir, time_below, time_above     → fraction 0-1
//	 gmi, ea1c, cv                   → % as 0-100
//	 avg, sd, percentiles            → mg/dL
//	 count                           → event count
//	 time_duration                   → minutes
func ComputeAggregate(agg Aggregate, args map[string]any, filter *Filter, in EvalInput) float64 {
	loc := in.TZ
	if loc == nil {
		loc = time.UTC
	}
	// Apply filter first — if a filter is set, the sample is counted only
	// when every specified field matches.
	match := func(s Sample) bool {
		if filter == nil {
			return true
		}
		if filter.GlucoseLt != nil && !(s.Value < *filter.GlucoseLt) {
			return false
		}
		if filter.GlucoseGt != nil && !(s.Value > *filter.GlucoseGt) {
			return false
		}
		if filter.Daypart != "" && filter.Daypart != DaypartAny {
			h := s.At.In(loc).Hour()
			switch filter.Daypart {
			case DaypartNight:
				if h >= 6 {
					return false
				}
			case DaypartMorning:
				if h < 6 || h >= 12 {
					return false
				}
			case DaypartAfternoon:
				if h < 12 || h >= 18 {
					return false
				}
			case DaypartEvening:
				if h < 18 {
					return false
				}
			}
		}
		return true
	}

	switch agg {
	case AggTIR:
		low, high := tirRange(args, 70, 180)
		return fractionInRange(in.Samples, match, low, high)
	case AggTimeBelow:
		t := floatArg(args, "threshold", 70)
		return fractionWhere(in.Samples, match, func(v float64) bool { return v < t })
	case AggTimeAbove:
		t := floatArg(args, "threshold", 180)
		return fractionWhere(in.Samples, match, func(v float64) bool { return v > t })
	case AggAvg:
		return meanFiltered(in.Samples, match)
	case AggSD:
		_, sd := meanSDFiltered(in.Samples, match)
		return sd
	case AggCV:
		mean, sd := meanSDFiltered(in.Samples, match)
		if mean <= 0 {
			return 0
		}
		return sd / mean * 100
	case AggGMI:
		mean := meanFiltered(in.Samples, match)
		if mean <= 0 {
			return 0
		}
		return 3.31 + 0.02392*mean
	case AggEA1C:
		mean := meanFiltered(in.Samples, match)
		if mean <= 0 {
			return 0
		}
		return (mean + 46.7) / 28.7
	case AggP10:
		return percentile(in.Samples, match, 10)
	case AggP25:
		return percentile(in.Samples, match, 25)
	case AggP50:
		return percentile(in.Samples, match, 50)
	case AggP75:
		return percentile(in.Samples, match, 75)
	case AggP90:
		return percentile(in.Samples, match, 90)
	case AggCount:
		c := 0
		for _, s := range in.Samples {
			if match(s) {
				c++
			}
		}
		return float64(c)
	case AggTimeDuration:
		// Each matching sample represents a 5-minute slice of data time.
		c := 0
		for _, s := range in.Samples {
			if match(s) {
				c++
			}
		}
		return float64(c) * 5
	default:
		return 0
	}
}

// AggregateUnit returns the string unit for display.
func AggregateUnit(agg Aggregate) string {
	switch agg {
	case AggTIR, AggTimeBelow, AggTimeAbove, AggCV, AggGMI, AggEA1C:
		return "%"
	case AggAvg, AggSD, AggP10, AggP25, AggP50, AggP75, AggP90:
		return "mg/dL"
	case AggCount:
		return "events"
	case AggTimeDuration:
		return "min"
	default:
		return ""
	}
}

// AggregateLabel is a human-friendly name for error / narrative text.
func AggregateLabel(agg Aggregate) string {
	switch agg {
	case AggTIR:
		return "Time in range"
	case AggTimeBelow:
		return "Time below threshold"
	case AggTimeAbove:
		return "Time above threshold"
	case AggGMI:
		return "GMI"
	case AggEA1C:
		return "eA1C"
	case AggAvg:
		return "Average glucose"
	case AggCV:
		return "Coefficient of variation"
	case AggSD:
		return "Standard deviation"
	case AggP10:
		return "10th percentile"
	case AggP25:
		return "25th percentile"
	case AggP50:
		return "Median glucose"
	case AggP75:
		return "75th percentile"
	case AggP90:
		return "90th percentile"
	case AggCount:
		return "Event count"
	case AggTimeDuration:
		return "Time in state"
	default:
		return string(agg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func tirRange(args map[string]any, defLow, defHigh float64) (float64, float64) {
	return floatArg(args, "low", defLow), floatArg(args, "high", defHigh)
}

func floatArg(args map[string]any, key string, fallback float64) float64 {
	if args == nil {
		return fallback
	}
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return fallback
}

func fractionInRange(samples []Sample, match func(Sample) bool, low, high float64) float64 {
	n, inRange := 0, 0
	for _, s := range samples {
		if !match(s) {
			continue
		}
		n++
		if s.Value >= low && s.Value <= high {
			inRange++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(inRange) / float64(n)
}

func fractionWhere(samples []Sample, match func(Sample) bool, pred func(float64) bool) float64 {
	n, hit := 0, 0
	for _, s := range samples {
		if !match(s) {
			continue
		}
		n++
		if pred(s.Value) {
			hit++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(hit) / float64(n)
}

func meanFiltered(samples []Sample, match func(Sample) bool) float64 {
	var sum float64
	n := 0
	for _, s := range samples {
		if !match(s) {
			continue
		}
		sum += s.Value
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func meanSDFiltered(samples []Sample, match func(Sample) bool) (float64, float64) {
	vals := make([]float64, 0, len(samples))
	var sum float64
	for _, s := range samples {
		if !match(s) {
			continue
		}
		vals = append(vals, s.Value)
		sum += s.Value
	}
	if len(vals) == 0 {
		return 0, 0
	}
	mean := sum / float64(len(vals))
	var variance float64
	for _, v := range vals {
		d := v - mean
		variance += d * d
	}
	return mean, math.Sqrt(variance / float64(len(vals)))
}

func percentile(samples []Sample, match func(Sample) bool, p float64) float64 {
	vals := make([]float64, 0, len(samples))
	for _, s := range samples {
		if match(s) {
			vals = append(vals, s.Value)
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	if len(vals) == 1 {
		return vals[0]
	}
	pos := (p / 100) * float64(len(vals)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return vals[lo]
	}
	return vals[lo] + (vals[hi]-vals[lo])*(pos-float64(lo))
}
