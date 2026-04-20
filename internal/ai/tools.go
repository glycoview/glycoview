package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/glycoview/glycoview/internal/ui"
)

// ToolHandler runs a tool against already-typed arguments and returns a
// JSON-serialisable result. Errors bubble up to the caller which records
// them and feeds them back to the model so it can recover.
type ToolHandler func(ctx context.Context, deps Deps, args json.RawMessage) (any, error)

type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"parameters"`
	Handler     ToolHandler    `json:"-"`
}

// Deps bundles the read-only services the tools consume. The AI never writes
// — it only reads and computes.
type Deps struct {
	UI ui.Service
}

// Registry returns the canonical, ordered list of tools exposed to the model.
func Registry() []ToolDef {
	return []ToolDef{
		toolNow(),
		toolComputeTIR(),
		toolComputeStats(),
		toolComputeAGP(),
		toolGetGlucose(),
		toolGetTreatments(),
		toolListExcursions(),
		toolGetLatestStatus(),
		toolGetProfile(),
	}
}

/* ──────────── argument types ──────────── */

type rangeArgs struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

type daysArgs struct {
	Days int `json:"days"`
}

type excursionArgs struct {
	StartMs   int64  `json:"startMs"`
	EndMs     int64  `json:"endMs"`
	Direction string `json:"direction"` // "low" | "high" | "severe_low" | "very_high"
}

/* ──────────── tool implementations ──────────── */

func toolNow() ToolDef {
	return ToolDef{
		Name:        "now",
		Description: "Returns the current server time as an ISO-8601 string and unix milliseconds. Always call this first so you know what 'today' means.",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ Deps, _ json.RawMessage) (any, error) {
			now := time.Now().UTC()
			return map[string]any{
				"iso":  now.Format(time.RFC3339),
				"ms":   now.UnixMilli(),
				"tz":   "UTC",
				"note": "All other tools accept unix-millisecond timestamps. Use this as the reference time.",
			}, nil
		},
	}
}

func toolComputeTIR() ToolDef {
	return ToolDef{
		Name: "compute_tir",
		Description: "Computes Time-In-Range over the given window. Returns the five clinical bands (<54, 54-70, 70-180, 180-250, >250) with percent and minutes. Also returns the tight-range 70-140% share.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"startMs": map[string]any{"type": "integer", "description": "Start of window, unix ms."},
				"endMs":   map[string]any{"type": "integer", "description": "End of window, unix ms."},
			},
			"required": []string{"startMs", "endMs"},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var args rangeArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			start, end, err := parseRange(args.StartMs, args.EndMs)
			if err != nil {
				return nil, err
			}
			daily, err := deps.UI.Daily(ctx, start)
			_ = daily
			// Call into ui.Service for TIR across the window
			return computeTIRFromService(ctx, deps, start, end)
		},
	}
}

func toolComputeStats() ToolDef {
	return ToolDef{
		Name: "compute_stats",
		Description: "Computes summary glucose statistics over the window: count, mean (mg/dL), standard deviation, coefficient of variation (CV%), GMI (Bergenstal 2018), and classic eA1C (ADAG).",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"startMs": map[string]any{"type": "integer"},
				"endMs":   map[string]any{"type": "integer"},
			},
			"required": []string{"startMs", "endMs"},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var args rangeArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			start, end, err := parseRange(args.StartMs, args.EndMs)
			if err != nil {
				return nil, err
			}
			resp, err := deps.UI.Trends(ctx, end, daysBetween(start, end))
			if err != nil {
				return nil, err
			}
			// Collect readings across days via aggregated daysSummary mean
			// (backend computes the overall mean in the `avg` metric).
			var avg, cv, sensor float64
			for _, m := range resp.Metrics {
				switch m.ID {
				case "avg":
					avg = firstNumber(m.Value)
				case "cv":
					cv = firstNumber(m.Value)
				case "sensor":
					sensor = firstNumber(m.Value)
				}
			}
			gmi := 3.31 + 0.02392*avg
			ea1c := (avg + 46.7) / 28.7
			return map[string]any{
				"startMs":           args.StartMs,
				"endMs":             args.EndMs,
				"avgGlucoseMgDl":    roundTo(avg, 1),
				"gmiPct":            roundTo(gmi, 2),
				"ea1cPctAdag":       roundTo(ea1c, 2),
				"cvPct":             roundTo(cv, 1),
				"sensorUsagePct":    roundTo(sensor, 1),
				"readingCountTotal": sumReadings(resp),
				"daysCovered":       len(resp.DaysSummary),
			}, nil
		},
	}
}

func toolComputeAGP() ToolDef {
	return ToolDef{
		Name:        "compute_agp",
		Description: "Returns the Ambulatory Glucose Profile — per-hour percentile buckets (p10, p25, p50, p75, p90, points) over the last N days. Use this when asked about patterns across the day.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"days": map[string]any{"type": "integer", "description": "Number of days to aggregate, default 14, max 90."},
			},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var args daysArgs
			_ = json.Unmarshal(raw, &args)
			if args.Days <= 0 {
				args.Days = 14
			}
			if args.Days > 90 {
				args.Days = 90
			}
			resp, err := deps.UI.Trends(ctx, time.Now().UTC(), args.Days)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"days":    args.Days,
				"buckets": resp.AGP,
				"tir":     resp.TimeInRange,
			}, nil
		},
	}
}

func toolGetGlucose() ToolDef {
	return ToolDef{
		Name:        "get_glucose",
		Description: "Returns glucose readings in a window. Large windows can return thousands of rows — prefer compute_stats / compute_agp for aggregates.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"startMs": map[string]any{"type": "integer"},
				"endMs":   map[string]any{"type": "integer"},
			},
			"required": []string{"startMs", "endMs"},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var args rangeArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			start, _, err := parseRange(args.StartMs, args.EndMs)
			if err != nil {
				return nil, err
			}
			resp, err := deps.UI.Daily(ctx, start)
			if err != nil {
				return nil, err
			}
			// Daily() returns a single-day window; clients that want multi-day
			// can iterate. For the AI we cap at one day per call.
			pts := filterGlucose(resp.Glucose, args.StartMs, args.EndMs)
			return map[string]any{
				"startMs":  args.StartMs,
				"endMs":    args.EndMs,
				"count":    len(pts),
				"readings": pts,
			}, nil
		},
	}
}

func toolGetTreatments() ToolDef {
	return ToolDef{
		Name:        "get_treatments",
		Description: "Returns treatment events (bolus, SMB, carbs, temp-basal, fingerstick) in a window. Each event has at (unix ms), kind, value (U for insulin, g for carbs, U/h for temp-basal) and optional duration.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"startMs": map[string]any{"type": "integer"},
				"endMs":   map[string]any{"type": "integer"},
			},
			"required": []string{"startMs", "endMs"},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var args rangeArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			start, _, err := parseRange(args.StartMs, args.EndMs)
			if err != nil {
				return nil, err
			}
			resp, err := deps.UI.Daily(ctx, start)
			if err != nil {
				return nil, err
			}
			carbs := filterEvents(resp.Carbs, args.StartMs, args.EndMs)
			bol := filterEvents(resp.Boluses, args.StartMs, args.EndMs)
			smb := filterEvents(resp.SMBs, args.StartMs, args.EndMs)
			tb := filterEvents(resp.TempBasals, args.StartMs, args.EndMs)
			smbg := filterEvents(resp.SMBGs, args.StartMs, args.EndMs)
			return map[string]any{
				"startMs":      args.StartMs,
				"endMs":        args.EndMs,
				"carbs":        carbs,
				"boluses":      bol,
				"smbs":         smb,
				"tempBasals":   tb,
				"fingersticks": smbg,
			}, nil
		},
	}
}

func toolListExcursions() ToolDef {
	return ToolDef{
		Name: "list_excursions",
		Description: "Lists glucose excursion events in a window. direction can be 'low' (<70), 'severe_low' (<54), 'high' (>180), or 'very_high' (>250). Each excursion has start, end, duration minutes, minimum/maximum, mean.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"startMs":   map[string]any{"type": "integer"},
				"endMs":     map[string]any{"type": "integer"},
				"direction": map[string]any{"type": "string", "enum": []string{"low", "severe_low", "high", "very_high"}},
			},
			"required": []string{"startMs", "endMs", "direction"},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var args excursionArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			start, _, err := parseRange(args.StartMs, args.EndMs)
			if err != nil {
				return nil, err
			}
			resp, err := deps.UI.Daily(ctx, start)
			if err != nil {
				return nil, err
			}
			pts := filterGlucose(resp.Glucose, args.StartMs, args.EndMs)
			return map[string]any{
				"direction":   args.Direction,
				"excursions":  findExcursions(pts, args.Direction),
				"totalChecks": len(pts),
			}, nil
		},
	}
}

func toolGetLatestStatus() ToolDef {
	return ToolDef{
		Name:        "get_latest_status",
		Description: "Returns the most recent pump-loop (Trio/OpenAPS) status: current IOB, COB, enacted temp basal rate, suggested correction reason, reservoir and pump battery. Use this for 'what's happening right now' questions.",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(ctx context.Context, deps Deps, _ json.RawMessage) (any, error) {
			resp, err := deps.UI.Devices(ctx, time.Now().UTC())
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"generatedAt": resp.GeneratedAt,
				"devices":     resp.Cards,
				"activity":    resp.Activity,
				"metrics":     resp.Metrics,
			}, nil
		},
	}
}

func toolGetProfile() ToolDef {
	return ToolDef{
		Name:        "get_profile",
		Description: "Returns the therapy profile (basal schedule, carb ratios, insulin sensitivities, targets). On Trio/OpenAPS appliances without a Nightscout profile upload, these are derived from the latest enacted loop status.",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(ctx context.Context, deps Deps, _ json.RawMessage) (any, error) {
			resp, err := deps.UI.Profile(ctx)
			if err != nil {
				return nil, err
			}
			return resp, nil
		},
	}
}

/* ──────────── computation helpers ──────────── */

func parseRange(startMs, endMs int64) (time.Time, time.Time, error) {
	if endMs <= startMs {
		return time.Time{}, time.Time{}, fmt.Errorf("endMs (%d) must be greater than startMs (%d)", endMs, startMs)
	}
	return time.UnixMilli(startMs).UTC(), time.UnixMilli(endMs).UTC(), nil
}

func daysBetween(start, end time.Time) int {
	span := end.Sub(start).Hours() / 24
	days := int(math.Ceil(span))
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}
	return days
}

func roundTo(v float64, decimals int) float64 {
	f := math.Pow10(decimals)
	return math.Round(v*f) / f
}

func firstNumber(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

func sumReadings(r ui.TrendsResponse) int {
	total := 0
	for _, b := range r.AGP {
		total += b.Points
	}
	return total
}

func computeTIRFromService(ctx context.Context, deps Deps, start, end time.Time) (any, error) {
	days := daysBetween(start, end)
	resp, err := deps.UI.Trends(ctx, end, days)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"startMs":     start.UnixMilli(),
		"endMs":       end.UnixMilli(),
		"daysCovered": len(resp.DaysSummary),
		"bands":       resp.TimeInRange,
	}, nil
}

func filterGlucose(pts []ui.GlucosePoint, startMs, endMs int64) []ui.GlucosePoint {
	out := make([]ui.GlucosePoint, 0, len(pts))
	for _, p := range pts {
		if p.At >= startMs && p.At <= endMs {
			out = append(out, p)
		}
	}
	return out
}

func filterEvents(events []ui.EventPoint, startMs, endMs int64) []ui.EventPoint {
	out := make([]ui.EventPoint, 0, len(events))
	for _, e := range events {
		if e.At >= startMs && e.At <= endMs {
			out = append(out, e)
		}
	}
	return out
}

type excursion struct {
	StartMs  int64   `json:"startMs"`
	EndMs    int64   `json:"endMs"`
	Minutes  int     `json:"minutes"`
	MinValue float64 `json:"minValue"`
	MaxValue float64 `json:"maxValue"`
	MeanValue float64 `json:"meanValue"`
}

func findExcursions(pts []ui.GlucosePoint, direction string) []excursion {
	if len(pts) == 0 {
		return nil
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].At < pts[j].At })
	var inRange func(v float64) bool
	switch direction {
	case "severe_low":
		inRange = func(v float64) bool { return v < 54 }
	case "high":
		inRange = func(v float64) bool { return v > 180 }
	case "very_high":
		inRange = func(v float64) bool { return v > 250 }
	default: // "low"
		inRange = func(v float64) bool { return v < 70 }
	}
	out := make([]excursion, 0)
	var cur *excursion
	var sum float64
	var n int
	for _, p := range pts {
		if inRange(p.Value) {
			if cur == nil {
				cur = &excursion{StartMs: p.At, MinValue: p.Value, MaxValue: p.Value}
				sum, n = 0, 0
			}
			cur.EndMs = p.At
			if p.Value < cur.MinValue {
				cur.MinValue = p.Value
			}
			if p.Value > cur.MaxValue {
				cur.MaxValue = p.Value
			}
			sum += p.Value
			n++
		} else if cur != nil {
			cur.Minutes = int((cur.EndMs - cur.StartMs) / 60000)
			if n > 0 {
				cur.MeanValue = roundTo(sum/float64(n), 0)
			}
			out = append(out, *cur)
			cur = nil
		}
	}
	if cur != nil {
		cur.Minutes = int((cur.EndMs - cur.StartMs) / 60000)
		if n > 0 {
			cur.MeanValue = roundTo(sum/float64(n), 0)
		}
		out = append(out, *cur)
	}
	return out
}
