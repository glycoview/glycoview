package ui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/glycoview/glycoview/internal/config"
	"github.com/glycoview/glycoview/internal/model"
	"github.com/glycoview/glycoview/internal/store"
)

type Service struct {
	Config config.Config
	Store  store.Store
}

func (s Service) Overview(ctx context.Context, now time.Time, days int) (OverviewResponse, error) {
	if days <= 0 {
		days = 14
	}
	entries, err := s.loadEntries(ctx, now.AddDate(0, 0, -days), now)
	if err != nil {
		return OverviewResponse{}, err
	}
	treatments, err := s.loadTreatments(ctx, now.AddDate(0, 0, -days), now)
	if err != nil {
		return OverviewResponse{}, err
	}
	statuses, err := s.loadStatuses(ctx, now.AddDate(0, 0, -7), now)
	if err != nil {
		return OverviewResponse{}, err
	}
	profile, _ := s.latestProfile(ctx)

	samples := glucoseSamples(entries)
	current := latestSample(samples)
	tir := timeInRange(samples)
	narrow := narrowTimeInRange(samples)
	carbs, insulin := totalsFromTreatments(treatments)

	return OverviewResponse{
		GeneratedAt: now.UnixMilli(),
		PatientName: patientName(profile),
		Subtitle:    fmt.Sprintf("%d day review window", days),
		Current: Metric{
			ID:      "glucose",
			Label:   "Current glucose",
			Value:   glucoseMetricValue(current),
			Detail:  glucoseDetail(current),
			Accent:  glucoseAccent(current.Value),
			Warning: current.Value < 70 || current.Value > 250,
		},
		Sparkline:   filterRecentSamples(samples, now.Add(-12*time.Hour)),
		TimeInRange: tir,
		NarrowRange: narrow,
		Metrics: []Metric{
			{ID: "avg", Label: "Average glucose", Value: fmt.Sprintf("%.0f mg/dL", averageGlucose(samples)), Accent: "cool"},
			{ID: "carbs", Label: "Documented carbs", Value: fmt.Sprintf("%.0f g", carbs), Accent: "amber"},
			{ID: "insulin", Label: "Documented insulin", Value: fmt.Sprintf("%.1f U", insulin), Accent: "blue"},
			{ID: "sensor", Label: "Sensor usage", Value: fmt.Sprintf("%.0f%%", sensorUsage(samples, days)), Accent: "green"},
		},
		Devices:  buildDeviceCards(statuses),
		Activity: buildRecentActivity(treatments, statuses, 7),
	}, nil
}

func (s Service) Daily(ctx context.Context, day time.Time) (DailyResponse, error) {
	day = day.UTC()
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	entries, err := s.loadEntries(ctx, start, end)
	if err != nil {
		return DailyResponse{}, err
	}
	treatments, err := s.loadTreatments(ctx, start, end)
	if err != nil {
		return DailyResponse{}, err
	}
	statuses, err := s.loadStatuses(ctx, start.Add(-6*time.Hour), end)
	if err != nil {
		return DailyResponse{}, err
	}
	profile, _ := s.latestProfile(ctx)

	samples := glucoseSamples(entries)
	split := splitTreatments(treatments)
	totalCarbs, totalInsulin := totalsFromTreatments(treatments)

	return DailyResponse{
		GeneratedAt:  day.UnixMilli(),
		PatientName:  patientName(profile),
		DateLabel:    start.Format("Monday, 02 Jan 2006"),
		RangeStart:   start.UnixMilli(),
		RangeEnd:     end.UnixMilli(),
		Glucose:      samples,
		Carbs:        split.Carbs,
		Insulin:      split.Insulin,
		Boluses:      split.Boluses,
		SMBs:         split.SMBs,
		TempBasals:   split.TempBasals,
		SMBGs:        split.SMBGs,
		BasalProfile: buildBasalProfile(profile, start, statuses),
		TimeInRange:  timeInRange(samples),
		Metrics: []Metric{
			{ID: "avg", Label: "Average glucose", Value: fmt.Sprintf("%.0f mg/dL", averageGlucose(samples)), Accent: "cool"},
			{ID: "carbs", Label: "Carbs today", Value: fmt.Sprintf("%.0f g", totalCarbs), Accent: "amber"},
			{ID: "insulin", Label: "Insulin today", Value: fmt.Sprintf("%.1f U", totalInsulin), Accent: "blue"},
			{ID: "events", Label: "Treatment events", Value: strconv.Itoa(len(treatments)), Accent: "rose"},
		},
		Devices: buildDeviceCards(statuses),
	}, nil
}

func (s Service) Trends(ctx context.Context, now time.Time, days int) (TrendsResponse, error) {
	if days <= 0 {
		days = 14
	}
	start := now.AddDate(0, 0, -days)
	entries, err := s.loadEntries(ctx, start, now)
	if err != nil {
		return TrendsResponse{}, err
	}
	treatments, err := s.loadTreatments(ctx, start, now)
	if err != nil {
		return TrendsResponse{}, err
	}
	profile, _ := s.latestProfile(ctx)
	samples := glucoseSamples(entries)

	return TrendsResponse{
		GeneratedAt: now.UnixMilli(),
		PatientName: patientName(profile),
		RangeLabel:  fmt.Sprintf("Last %d days", days),
		Days:        days,
		AGP:         buildAGP(samples),
		TimeInRange: timeInRange(samples),
		Metrics: []Metric{
			{ID: "avg", Label: "Average glucose", Value: fmt.Sprintf("%.0f mg/dL", averageGlucose(samples)), Accent: "cool"},
			{ID: "cv", Label: "Coefficient of variation", Value: fmt.Sprintf("%.0f%%", glucoseCV(samples)), Accent: "violet"},
			{ID: "sensor", Label: "Sensor usage", Value: fmt.Sprintf("%.0f%%", sensorUsage(samples, days)), Accent: "green"},
			{ID: "days", Label: "Days analysed", Value: strconv.Itoa(days), Accent: "slate"},
		},
		DaysSummary: buildDailySummaries(samples, treatments),
	}, nil
}

func (s Service) Profile(ctx context.Context) (ProfileResponse, error) {
	profile, _ := s.latestProfile(ctx)
	hasProfile := profile.Identifier() != ""

	now := time.Now().UTC()
	statuses, _ := s.loadStatuses(ctx, now.Add(-24*time.Hour), now)

	if hasProfile {
		return ProfileResponse{
			GeneratedAt:   time.Now().UnixMilli(),
			PatientName:   patientName(profile),
			Headline:      profileHeadline(profile),
			Metrics:       buildProfileMetrics(profile),
			BasalSchedule: profileSchedule(profile, "basal"),
			CarbRatios:    profileSchedule(profile, "carbratio"),
			Sensitivity:   profileSchedule(profile, "sens"),
			Targets:       profileTargets(profile),
			Notes: []ActivityItem{
				{Title: "Therapy profile", Detail: "Latest profile imported from Nightscout-compatible storage", Kind: "profile", Accent: "blue"},
			},
		}, nil
	}

	// No Nightscout profile record — Trio/OpenAPS deployments don't always
	// upload one. Derive what we can from the latest pump-loop status.
	return deriveProfileFromStatuses(statuses), nil
}

func deriveProfileFromStatuses(statuses []model.Record) ProfileResponse {
	now := time.Now().UnixMilli()
	resp := ProfileResponse{
		GeneratedAt: now,
		PatientName: "Appliance owner",
		Headline:    "Derived from loop status — no Nightscout profile uploaded",
	}
	if len(statuses) == 0 {
		resp.Headline = "Waiting for pump loop data"
		return resp
	}

	// Pick the most recent status with an openaps payload.
	var latest model.Record
	for i := len(statuses) - 1; i >= 0; i-- {
		if model.PathValue(statuses[i].Data, "openaps") != nil {
			latest = statuses[i]
			break
		}
	}
	if latest.Identifier() == "" {
		latest = statuses[len(statuses)-1]
	}

	source := model.PathValue(latest.Data, "openaps.suggested")
	if _, ok := source.(map[string]any); !ok {
		source = model.PathValue(latest.Data, "openaps.enacted")
	}
	sourceMap, _ := source.(map[string]any)

	device, _ := model.StringField(latest.Data, "device")
	if device == "" {
		device = "Pump loop"
	}
	resp.Headline = device + " · live therapy settings"

	metricFor := func(id, label, path, unit string, precision int) (Metric, bool) {
		value, ok := floatValue(model.PathValue(sourceMap, path))
		if !ok {
			return Metric{}, false
		}
		format := "%." + strconv.Itoa(precision) + "f"
		text := fmt.Sprintf(format, value)
		if unit != "" {
			text = text + " " + unit
		}
		return Metric{ID: id, Label: label, Value: text, Accent: "cool"}, true
	}
	metrics := make([]Metric, 0, 8)
	if m, ok := metricFor("tdd", "TDD (loop)", "TDD", "U", 1); ok {
		metrics = append(metrics, m)
	}
	if m, ok := metricFor("cr", "Carb ratio", "CR", "g/U", 0); ok {
		metrics = append(metrics, m)
	}
	if m, ok := metricFor("isf", "Sensitivity", "ISF", "mg/dL/U", 0); ok {
		metrics = append(metrics, m)
	}
	if m, ok := metricFor("target", "Target", "current_target", "mg/dL", 0); ok {
		metrics = append(metrics, m)
	}
	if m, ok := metricFor("iob", "IOB", "IOB", "U", 2); ok {
		metrics = append(metrics, m)
	}
	if m, ok := metricFor("cob", "COB", "COB", "g", 0); ok {
		metrics = append(metrics, m)
	}
	if reservoir, ok := floatValue(model.PathValue(latest.Data, "pump.reservoir")); ok {
		metrics = append(metrics, Metric{ID: "reservoir", Label: "Reservoir", Value: fmt.Sprintf("%.1f U", reservoir), Accent: "blue"})
	}
	resp.Metrics = metrics

	// Single-point "schedules" built from whatever scalar values we have.
	pointOrNil := func(value any, format string) []SchedulePoint {
		f, ok := floatValue(value)
		if !ok {
			return nil
		}
		return []SchedulePoint{{Time: "00:00", Value: fmt.Sprintf(format, f)}}
	}
	if rate, ok := floatValue(model.PathValue(latest.Data, "openaps.enacted.rate")); ok {
		resp.BasalSchedule = []SchedulePoint{{Time: "00:00", Value: fmt.Sprintf("%.2f U/h", rate)}}
	}
	if rows := pointOrNil(model.PathValue(sourceMap, "CR"), "1:%.0f"); rows != nil {
		resp.CarbRatios = rows
	}
	if rows := pointOrNil(model.PathValue(sourceMap, "ISF"), "%.0f mg/dL/U"); rows != nil {
		resp.Sensitivity = rows
	}
	if rows := pointOrNil(model.PathValue(sourceMap, "current_target"), "%.0f mg/dL"); rows != nil {
		resp.Targets = rows
	}

	resp.Notes = []ActivityItem{
		{
			At:     latest.SrvModified,
			Title:  "Derived from pump loop",
			Detail: fmt.Sprintf("Values taken from the latest %s status — upload a Nightscout profile to override.", device),
			Kind:   "profile",
			Accent: "violet",
		},
	}

	return resp
}

func (s Service) Devices(ctx context.Context, now time.Time) (DevicesResponse, error) {
	statuses, err := s.loadStatuses(ctx, now.AddDate(0, 0, -14), now)
	if err != nil {
		return DevicesResponse{}, err
	}
	profile, _ := s.latestProfile(ctx)
	cards := buildDeviceCards(statuses)
	return DevicesResponse{
		GeneratedAt: now.UnixMilli(),
		PatientName: patientName(profile),
		Headline:    "Latest device and integration state",
		Cards:       cards,
		Metrics: []Metric{
			{ID: "count", Label: "Tracked device feeds", Value: strconv.Itoa(len(cards)), Accent: "blue"},
			{ID: "active", Label: "Feeds updated in 24h", Value: strconv.Itoa(countFreshDevices(cards, now)), Accent: "green"},
			{ID: "api", Label: "Server version", Value: s.Config.AppVersion, Accent: "violet"},
		},
		Activity: latestStatusActivity(statuses, 8),
	}, nil
}
