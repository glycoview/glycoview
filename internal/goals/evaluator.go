package goals

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Evaluate runs a predicate against a set of samples and returns the full
// Progress payload: current aggregate value, a daily value series for the
// chart, a linear trajectory, and an achievement state.
//
// windowEnd is "now" in the user's timezone; windowStart = windowEnd -
// predicate.Window.Days.
func Evaluate(pred Predicate, targetDate string, samples []Sample, tz *time.Location) Progress {
	if tz == nil {
		tz = time.UTC
	}

	days := pred.Window.Days
	if days <= 0 {
		days = 14
	}
	if days > 365 {
		days = 365
	}
	end := time.Now().In(tz)
	start := end.AddDate(0, 0, -days)

	// Slice samples to window.
	inWindow := make([]Sample, 0, len(samples))
	for _, s := range samples {
		if !s.At.Before(start) && !s.At.After(end) {
			inWindow = append(inWindow, s)
		}
	}

	unit := AggregateUnit(pred.Aggregate)

	// 1) Daily series — one value per local day over the window.
	dailySeries := buildDailySeries(pred, inWindow, start, end, tz)

	// 2) Current value — aggregate over the whole window.
	current := ComputeAggregate(pred.Aggregate, pred.Args, pred.Filter, EvalInput{
		Samples: inWindow,
		TZ:      tz,
	})

	// Mark "met" on each daily point.
	targetVal, _ := extractTarget(pred)
	for i := range dailySeries {
		dailySeries[i].Met = compareOp(dailySeries[i].Value, pred.Op, pred.Value, valPtr(pred.Value2))
	}

	met := compareOp(current, pred.Op, pred.Value, valPtr(pred.Value2))

	// 3) Optional per-unit bucket evaluation.
	var perUnitResult *PerUnitResult
	if pred.PerUnit != nil {
		perUnitResult = buildPerUnitResult(pred, inWindow, start, end, tz)
	}

	// 4) Trajectory — linear regression on daily series.
	var trajectory *Trajectory
	if len(dailySeries) >= 3 {
		slope, intercept := linearFit(dailySeries)
		trajectory = &Trajectory{SlopePerDay: slope}
		// Project to target date if provided.
		if targetDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", targetDate, tz); err == nil {
				daysAhead := math.Round(t.Sub(end).Hours() / 24)
				x := float64(len(dailySeries)-1) + daysAhead
				proj := slope*x + intercept
				trajectory.ProjectedAtTarget = &proj
				if compareOp(proj, pred.Op, pred.Value, valPtr(pred.Value2)) {
					// Projection meets target. Find the day we'd cross the threshold.
					if crossDays, ok := daysToTarget(slope, intercept, pred); ok {
						ahead := daysAhead - crossDays
						trajectory.DaysAheadOfSchedule = &ahead
					}
				}
			}
		}
	}

	// 5) State + narrative.
	state := classifyState(met, current, targetVal, trajectory, pred, targetDate)
	narrative, nudge := describe(pred, current, targetVal, state, trajectory, perUnitResult, unit)

	return Progress{
		CurrentValue: round2(current),
		TargetValue:  round2(pred.Value),
		Unit:         unit,
		Met:          met,
		State:        state,
		Narrative:    narrative,
		Nudge:        nudge,
		DailySeries:  dailySeries,
		Trajectory:   trajectory,
		PerUnit:      perUnitResult,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// building blocks
// ─────────────────────────────────────────────────────────────────────────────

func buildDailySeries(pred Predicate, samples []Sample, start, end time.Time, tz *time.Location) []DailyPoint {
	days := int(math.Ceil(end.Sub(start).Hours() / 24))
	if days < 1 {
		days = 1
	}
	// Bucket samples by local date.
	byDate := make(map[string][]Sample, days)
	for _, s := range samples {
		key := s.At.In(tz).Format("2006-01-02")
		byDate[key] = append(byDate[key], s)
	}
	out := make([]DailyPoint, 0, days)
	cur := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, tz)
	last := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, tz)
	for !cur.After(last) {
		key := cur.Format("2006-01-02")
		val := ComputeAggregate(pred.Aggregate, pred.Args, pred.Filter, EvalInput{
			Samples: byDate[key],
			TZ:      tz,
		})
		out = append(out, DailyPoint{
			Date:  key,
			Value: round2(val),
		})
		cur = cur.AddDate(0, 0, 1)
	}
	return out
}

func buildPerUnitResult(pred Predicate, samples []Sample, start, end time.Time, tz *time.Location) *PerUnitResult {
	if pred.PerUnit == nil {
		return nil
	}
	bucketDays := 7
	if pred.PerUnit.Kind == "day" {
		bucketDays = 1
	}
	// Buckets run backwards from end.
	buckets := make([]PerUnitBucket, 0)
	cursor := end
	for cursor.After(start) {
		bStart := cursor.AddDate(0, 0, -bucketDays)
		if bStart.Before(start) {
			bStart = start
		}
		in := make([]Sample, 0)
		for _, s := range samples {
			if !s.At.Before(bStart) && s.At.Before(cursor) {
				in = append(in, s)
			}
		}
		val := ComputeAggregate(pred.Aggregate, pred.Args, pred.Filter, EvalInput{
			Samples: in,
			TZ:      tz,
		})
		met := compareOp(val, pred.Op, pred.Value, valPtr(pred.Value2))
		label := bStart.Format("2006-01-02")
		if bucketDays == 7 {
			label = "Week of " + label
		}
		buckets = append([]PerUnitBucket{{Label: label, Value: round2(val), Met: met}}, buckets...)
		cursor = bStart
	}
	metCount := 0
	for _, b := range buckets {
		if b.Met {
			metCount++
		}
	}
	return &PerUnitResult{
		Kind:       pred.PerUnit.Kind,
		Buckets:    buckets,
		MetCount:   metCount,
		TotalCount: len(buckets),
	}
}

func compareOp(v float64, op Operator, value float64, value2 *float64) bool {
	switch op {
	case OpGE:
		return v >= value
	case OpLE:
		return v <= value
	case OpGT:
		return v > value
	case OpLT:
		return v < value
	case OpEQ:
		return math.Abs(v-value) < 1e-9
	case OpBetween:
		if value2 == nil {
			return v >= value
		}
		return v >= value && v <= *value2
	}
	return false
}

func valPtr(v *float64) *float64 { return v }

// linearFit returns (slope per day index, intercept). Days are numbered 0..N-1.
func linearFit(series []DailyPoint) (float64, float64) {
	n := float64(len(series))
	if n < 2 {
		return 0, 0
	}
	var sumX, sumY, sumXY, sumXX float64
	for i, p := range series {
		x := float64(i)
		y := p.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	den := n*sumXX - sumX*sumX
	if den == 0 {
		return 0, sumY / n
	}
	slope := (n*sumXY - sumX*sumY) / den
	intercept := (sumY - slope*sumX) / n
	return slope, intercept
}

// daysToTarget returns the x-offset (days from series start) at which the
// linear projection first crosses the threshold in the op's direction.
func daysToTarget(slope, intercept float64, pred Predicate) (float64, bool) {
	if slope == 0 {
		return 0, false
	}
	x := (pred.Value - intercept) / slope
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0, false
	}
	return x, true
}

func extractTarget(pred Predicate) (float64, *float64) {
	return pred.Value, pred.Value2
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ─────────────────────────────────────────────────────────────────────────────
// narrative
// ─────────────────────────────────────────────────────────────────────────────

func classifyState(met bool, current, _ float64, traj *Trajectory, _ Predicate, targetDate string) ProgressState {
	if targetDate == "" {
		if met {
			return StateOnTrack
		}
		return StateOngoing
	}
	if met {
		// Check margin for "smashing". Smashing = beating target by ≥ 5 pp (if % aggregate) or 5 units (otherwise).
		// Keep the simple rule: if trajectory is stable or improving AND we're met, call it smashing if the gap is wide.
		return StateOnTrack
	}
	// Not met. Use trajectory to classify.
	if traj != nil && traj.ProjectedAtTarget != nil {
		_ = current
		// on_track if projection would meet target
		// we'd ideally check compareOp(proj, pred.Op, pred.Value) but the
		// caller already baked that into trajectory.DaysAheadOfSchedule
		if traj.DaysAheadOfSchedule != nil {
			return StateOnTrack
		}
	}
	if traj != nil && traj.SlopePerDay != 0 {
		// Heuristic: if slope is moving toward target (sign of slope matches op direction),
		// call it at_risk. Otherwise behind.
		// For now keep it simple — any non-null slope with active goal is at_risk when not met.
		return StateAtRisk
	}
	return StateBehind
}

func describe(pred Predicate, current, target float64, state ProgressState, traj *Trajectory, per *PerUnitResult, unit string) (string, string) {
	label := AggregateLabel(pred.Aggregate)
	cur := formatValue(current, unit, pred.Aggregate)
	tgt := formatValue(target, unit, pred.Aggregate)
	opWord := opPhrase(pred.Op)

	switch state {
	case StateAchieved:
		return fmt.Sprintf("Achieved — %s %s %s.", label, opWord, tgt), ""
	case StateSmashing:
		return fmt.Sprintf("Smashing — %s at %s, target %s.", label, cur, tgt), ""
	case StateOngoing:
		if per != nil {
			return fmt.Sprintf("%s is %s; target %s %s. %d of %d %s met.",
				label, cur, opWord, tgt, per.MetCount, per.TotalCount, bucketWord(per.Kind, per.TotalCount)), ""
		}
		return fmt.Sprintf("%s is %s; target %s %s.", label, cur, opWord, tgt), ""
	case StateOnTrack:
		if traj != nil && traj.DaysAheadOfSchedule != nil {
			ahead := int(math.Round(*traj.DaysAheadOfSchedule))
			if ahead > 0 {
				return fmt.Sprintf("On track — %s at %s, target %s %s. %d days ahead of schedule.", label, cur, opWord, tgt, ahead), ""
			}
		}
		return fmt.Sprintf("On track — %s at %s, target %s %s.", label, cur, opWord, tgt), ""
	case StateAtRisk:
		gap := math.Abs(target - current)
		return fmt.Sprintf("At risk — %s at %s, %s short of %s.", label, cur, formatGap(gap, unit, pred.Aggregate), tgt),
			atRiskNudge(pred, current, target)
	case StateBehind:
		return fmt.Sprintf("Behind — %s at %s, target %s %s.", label, cur, opWord, tgt),
			"Revisit your action plan with the care team."
	}
	return "", ""
}

func bucketWord(kind string, n int) string {
	base := "week"
	if kind == "day" {
		base = "day"
	}
	if n == 1 {
		return base
	}
	return base + "s"
}

func atRiskNudge(pred Predicate, current, target float64) string {
	switch pred.Aggregate {
	case AggTIR:
		return "Aim for a few more in-range hours this week — pre-bolus meals if glucose is rising."
	case AggTimeBelow:
		return "Dial back late-evening corrections; log any hypo events for your clinician."
	case AggTimeAbove:
		return "Watch post-meal spikes; consider adjusting I:C ratio with your doctor."
	case AggGMI, AggEA1C:
		if target < current {
			return "Small steady TIR gains translate into GMI drops over weeks."
		}
		return ""
	case AggCount:
		return "Review the events around the thresholds to understand what's driving them."
	}
	return ""
}

func opPhrase(op Operator) string {
	switch op {
	case OpGE:
		return "at or above"
	case OpLE:
		return "at or below"
	case OpGT:
		return "above"
	case OpLT:
		return "below"
	case OpEQ:
		return "equal to"
	case OpBetween:
		return "between"
	}
	return string(op)
}

func formatValue(v float64, unit string, agg Aggregate) string {
	// Fractional aggregates come out of the evaluator as 0-1; render as percent.
	if unit == "%" && isFraction(agg) {
		return fmt.Sprintf("%.1f%%", v*100)
	}
	if unit == "%" {
		return fmt.Sprintf("%.1f%%", v)
	}
	if unit == "mg/dL" {
		return fmt.Sprintf("%.0f mg/dL", v)
	}
	if unit == "events" {
		return strings.TrimSuffix(fmt.Sprintf("%.1f events", v), ".0 events") + " events"
	}
	if unit == "min" {
		return fmt.Sprintf("%.0f min", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func isFraction(a Aggregate) bool {
	switch a {
	case AggTIR, AggTimeBelow, AggTimeAbove:
		return true
	}
	return false
}

func formatGap(gap float64, unit string, agg Aggregate) string {
	if unit == "%" && isFraction(agg) {
		return fmt.Sprintf("%.1f pp", gap*100)
	}
	if unit == "%" {
		return fmt.Sprintf("%.1f pp", gap)
	}
	if unit == "mg/dL" {
		return fmt.Sprintf("%.0f mg/dL", gap)
	}
	return fmt.Sprintf("%.2f", gap)
}
