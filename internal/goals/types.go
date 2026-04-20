package goals

import (
	"encoding/json"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Predicate — the math
//
// Every goal is a mathematical predicate that should be true (or trend toward
// true) by an optional target date. Predicates are a small closed language so
// both the evaluator and the AI can reason about them symbolically. No
// free-form expressions, no parsing.
// ─────────────────────────────────────────────────────────────────────────────

type Aggregate string

const (
	AggTIR          Aggregate = "tir"
	AggTimeBelow    Aggregate = "time_below"
	AggTimeAbove    Aggregate = "time_above"
	AggGMI          Aggregate = "gmi"
	AggEA1C         Aggregate = "ea1c"
	AggAvg          Aggregate = "avg"
	AggCV           Aggregate = "cv"
	AggSD           Aggregate = "sd"
	AggP10          Aggregate = "p10"
	AggP25          Aggregate = "p25"
	AggP50          Aggregate = "p50"
	AggP75          Aggregate = "p75"
	AggP90          Aggregate = "p90"
	AggCount        Aggregate = "count"
	AggTimeDuration Aggregate = "time_duration"
)

type Operator string

const (
	OpGE      Operator = ">="
	OpLE      Operator = "<="
	OpGT      Operator = ">"
	OpLT      Operator = "<"
	OpEQ      Operator = "=="
	OpBetween Operator = "between"
)

// Daypart slices the day into 6-hour blocks in the user's timezone.
type Daypart string

const (
	DaypartAny       Daypart = ""
	DaypartNight     Daypart = "night"     // 00-06
	DaypartMorning   Daypart = "morning"   // 06-12
	DaypartAfternoon Daypart = "afternoon" // 12-18
	DaypartEvening   Daypart = "evening"   // 18-24
)

// Filter narrows what count / time_duration are counting.
// Multiple fields AND together.
type Filter struct {
	GlucoseLt *float64 `json:"glucoseLt,omitempty"`
	GlucoseGt *float64 `json:"glucoseGt,omitempty"`
	Daypart   Daypart  `json:"daypart,omitempty"`
	EventType string   `json:"eventType,omitempty"` // bolus|smb|carbs|temp_basal (v2)
}

type WindowKind string

const (
	WindowTrailingDays WindowKind = "trailing_days"
)

type Window struct {
	Kind WindowKind `json:"kind"`
	Days int        `json:"days"` // 1..365
}

// PerUnit turns a goal into "must hold on every <kind> of the window".
// Example: count(glucose<54) <= 2 per week / requireAll=true → the inner
// predicate must hold on each sub-week.
type PerUnit struct {
	Kind       string `json:"kind"` // day|week
	RequireAll bool   `json:"requireAll"`
}

// PredicateKind distinguishes between a direct threshold check and a
// trend-over-time requirement.
type PredicateKind string

const (
	KindThreshold PredicateKind = "threshold"
	KindTrend     PredicateKind = "trend"
)

// Args holds aggregate-specific parameters.
//   - tir            → {low, high}
//   - time_below     → {threshold}
//   - time_above     → {threshold}
//   - all others     → (none)
type Predicate struct {
	Kind          PredicateKind  `json:"kind"`
	Aggregate     Aggregate      `json:"aggregate"`
	Args          map[string]any `json:"args,omitempty"`
	Filter        *Filter        `json:"filter,omitempty"`
	Op            Operator       `json:"op"`
	Value         float64        `json:"value"`
	Value2        *float64       `json:"value2,omitempty"` // between(lo, hi): hi goes here
	Window        Window         `json:"window"`
	PerUnit       *PerUnit       `json:"perUnit,omitempty"`
	TrendOverDays int            `json:"trendOverDays,omitempty"` // trend kind only
}

// ─────────────────────────────────────────────────────────────────────────────
// Persisted shape
// ─────────────────────────────────────────────────────────────────────────────

type Status string

const (
	StatusActive   Status = "active"
	StatusAchieved Status = "achieved"
	StatusPaused   Status = "paused"
	StatusArchived Status = "archived"
)

type Goal struct {
	ID          string     `json:"id"`
	CreatedBy   string     `json:"createdBy,omitempty"`
	Title       string     `json:"title"`
	Predicate   Predicate  `json:"predicate"`
	StartDate   string     `json:"startDate"`
	TargetDate  string     `json:"targetDate,omitempty"` // empty = ongoing (no deadline)
	Status      Status     `json:"status"`
	Rationale   string     `json:"rationale,omitempty"`
	ActionPlan  string     `json:"actionPlan,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Progress
//
// Everything the frontend and Glyco need to render and reason about a goal
// after evaluating the predicate against the user's data.
// ─────────────────────────────────────────────────────────────────────────────

type ProgressState string

const (
	StateSmashing ProgressState = "smashing" // beating target by ≥ 5 pp (or equivalent)
	StateOnTrack  ProgressState = "on_track" // meeting now OR projection meets
	StateAtRisk   ProgressState = "at_risk"  // projection misses by < 20%
	StateBehind   ProgressState = "behind"   // projection misses by more
	StateAchieved ProgressState = "achieved" // status flipped
	StateOngoing  ProgressState = "ongoing"  // no target date; meeting-now only
)

type DailyPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
	Met   bool    `json:"met"`
}

type Trajectory struct {
	SlopePerDay         float64  `json:"slopePerDay"`
	ProjectedAtTarget   *float64 `json:"projectedAtTarget,omitempty"`
	DaysAheadOfSchedule *float64 `json:"daysAheadOfSchedule,omitempty"`
}

type PerUnitBucket struct {
	Label string  `json:"label"` // "Week of 2026-04-14"
	Value float64 `json:"value"`
	Met   bool    `json:"met"`
}

type PerUnitResult struct {
	Kind       string          `json:"kind"`
	Buckets    []PerUnitBucket `json:"buckets"`
	MetCount   int             `json:"metCount"`
	TotalCount int             `json:"totalCount"`
}

type Progress struct {
	CurrentValue float64        `json:"currentValue"`
	TargetValue  float64        `json:"targetValue"`
	Unit         string         `json:"unit"` // "%" | "mg/dL" | "events" | "min"
	Met          bool           `json:"met"`
	State        ProgressState  `json:"state"`
	Narrative    string         `json:"narrative"`   // 1 line of plain English
	Nudge        string         `json:"nudge,omitempty"`
	DailySeries  []DailyPoint   `json:"dailySeries"`
	Trajectory   *Trajectory    `json:"trajectory,omitempty"`
	PerUnit      *PerUnitResult `json:"perUnit,omitempty"`
}

// WithProgress attaches a Progress to a Goal for list responses.
type WithProgress struct {
	Goal
	Progress *Progress `json:"progress,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// JSON helpers — predicate <-> JSON
// ─────────────────────────────────────────────────────────────────────────────

func (p Predicate) Marshal() ([]byte, error) { return json.Marshal(p) }

func UnmarshalPredicate(raw []byte) (Predicate, error) {
	var p Predicate
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	return p, nil
}
