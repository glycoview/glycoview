package goals

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/glycoview/glycoview/internal/model"
	"github.com/glycoview/glycoview/internal/store"
)

// SampleSource fetches raw glucose samples in a time range. We depend on the
// shared nightscout-style store directly so goals don't couple to the UI
// service's higher-level response shapes.
type SampleSource interface {
	Samples(ctx context.Context, start, end time.Time) ([]Sample, error)
}

// NightscoutStoreSource reads entries out of the shared store.
type NightscoutStoreSource struct {
	Store store.Store
}

// Samples returns SGV/mbg/glucose readings in [start, end] converted to the
// local Sample shape for the evaluator. Matches the behaviour of
// internal/ui.glucoseSamples.
func (s NightscoutStoreSource) Samples(ctx context.Context, start, end time.Time) ([]Sample, error) {
	records, err := s.Store.Search(ctx, "entries", store.Query{
		Filters: []store.Filter{
			{Field: "date", Op: "gte", Value: itoa(start.UnixMilli())},
			{Field: "date", Op: "lte", Value: itoa(end.UnixMilli())},
		},
		Limit:     200000,
		SortField: "date",
		SortDesc:  false,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Sample, 0, len(records))
	for _, r := range records {
		value, ok := glucoseValue(r.Data)
		if !ok {
			continue
		}
		date, ok := model.Int64Field(r.Data, "date")
		if !ok {
			continue
		}
		out = append(out, Sample{
			At:    time.UnixMilli(date).UTC(),
			Value: value,
		})
	}
	return out, nil
}

func glucoseValue(data map[string]any) (float64, bool) {
	for _, field := range []string{"sgv", "mbg", "glucose"} {
		if value := model.PathValue(data, field); value != nil {
			switch n := value.(type) {
			case float64:
				return n, true
			case float32:
				return float64(n), true
			case int:
				return float64(n), true
			case int64:
				return float64(n), true
			}
		}
	}
	return 0, false
}

func itoa(v int64) string {
	return fmt.Sprintf("%d", v)
}

// ─────────────────────────────────────────────────────────────────────────────
// Service
// ─────────────────────────────────────────────────────────────────────────────

type Service struct {
	Store    Store
	Samples  SampleSource
	Location func() *time.Location // optional tz resolver; defaults to UTC
}

func (s Service) loc() *time.Location {
	if s.Location != nil {
		if l := s.Location(); l != nil {
			return l
		}
	}
	return time.UTC
}

func (s Service) List(ctx context.Context, includeArchived bool, tz *time.Location) ([]WithProgress, error) {
	goals, err := s.Store.List(ctx, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]WithProgress, 0, len(goals))
	for _, g := range goals {
		wp := WithProgress{Goal: g}
		if p, err := s.evaluate(ctx, g, tz); err == nil {
			wp.Progress = p
		}
		out = append(out, wp)
	}
	return out, nil
}

func (s Service) Get(ctx context.Context, id string, tz *time.Location) (WithProgress, error) {
	g, err := s.Store.Get(ctx, id)
	if err != nil {
		return WithProgress{}, err
	}
	wp := WithProgress{Goal: g}
	if p, err := s.evaluate(ctx, g, tz); err == nil {
		wp.Progress = p
	}
	return wp, nil
}

// Preview evaluates a predicate without saving anything. Used for the live
// preview in the rule builder.
func (s Service) Preview(ctx context.Context, pred Predicate, targetDate string, tz *time.Location) (*Progress, error) {
	if tz == nil {
		tz = s.loc()
	}
	days := pred.Window.Days
	if days <= 0 {
		days = 14
	}
	end := time.Now().In(tz)
	start := end.AddDate(0, 0, -days)
	samples, err := s.Samples.Samples(ctx, start, end)
	if err != nil {
		return nil, err
	}
	p := Evaluate(pred, targetDate, samples, tz)
	return &p, nil
}

func (s Service) Create(ctx context.Context, g Goal) (WithProgress, error) {
	if err := validate(g); err != nil {
		return WithProgress{}, err
	}
	created, err := s.Store.Create(ctx, g)
	if err != nil {
		return WithProgress{}, err
	}
	return s.Get(ctx, created.ID, nil)
}

func (s Service) Update(ctx context.Context, g Goal) (WithProgress, error) {
	if err := validate(g); err != nil {
		return WithProgress{}, err
	}
	updated, err := s.Store.Update(ctx, g)
	if err != nil {
		return WithProgress{}, err
	}
	return s.Get(ctx, updated.ID, nil)
}

func (s Service) SetStatus(ctx context.Context, id string, status Status) (WithProgress, error) {
	var completedAt *time.Time
	if status == StatusAchieved {
		n := time.Now().UTC()
		completedAt = &n
	}
	updated, err := s.Store.SetStatus(ctx, id, status, completedAt)
	if err != nil {
		return WithProgress{}, err
	}
	return s.Get(ctx, updated.ID, nil)
}

func (s Service) Delete(ctx context.Context, id string) error {
	return s.Store.Delete(ctx, id)
}

// evaluate wraps the evaluator + data fetch. Returns nil, error if the data
// fetch fails — caller can still return the goal without progress.
func (s Service) evaluate(ctx context.Context, g Goal, tz *time.Location) (*Progress, error) {
	if tz == nil {
		tz = s.loc()
	}
	days := g.Predicate.Window.Days
	if days <= 0 {
		days = 14
	}
	end := time.Now().In(tz)
	start := end.AddDate(0, 0, -days)
	samples, err := s.Samples.Samples(ctx, start, end)
	if err != nil {
		return nil, err
	}
	p := Evaluate(g.Predicate, g.TargetDate, samples, tz)
	if g.Status == StatusAchieved {
		p.State = StateAchieved
		p.Narrative = "Achieved."
	}
	return &p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// validation
// ─────────────────────────────────────────────────────────────────────────────

func validate(g Goal) error {
	if strings.TrimSpace(g.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if g.StartDate == "" {
		return fmt.Errorf("startDate is required")
	}
	pred := g.Predicate
	if pred.Aggregate == "" {
		return fmt.Errorf("predicate.aggregate is required")
	}
	if pred.Op == "" {
		return fmt.Errorf("predicate.op is required")
	}
	if pred.Window.Kind == "" {
		pred = Predicate{Kind: pred.Kind, Aggregate: pred.Aggregate, Args: pred.Args,
			Filter: pred.Filter, Op: pred.Op, Value: pred.Value, Value2: pred.Value2,
			Window: Window{Kind: WindowTrailingDays, Days: 14}, PerUnit: pred.PerUnit,
			TrendOverDays: pred.TrendOverDays}
	}
	return nil
}
