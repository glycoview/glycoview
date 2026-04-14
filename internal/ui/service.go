package ui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/glycoview/glycoview/internal/config"
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
	carbs, insulinEvents := splitTreatmentEvents(treatments)
	totalCarbs, totalInsulin := totalsFromTreatments(treatments)

	return DailyResponse{
		GeneratedAt:  day.UnixMilli(),
		PatientName:  patientName(profile),
		DateLabel:    start.Format("Monday, 02 Jan 2006"),
		RangeStart:   start.UnixMilli(),
		RangeEnd:     end.UnixMilli(),
		Glucose:      samples,
		Carbs:        carbs,
		Insulin:      insulinEvents,
		BasalProfile: buildBasalProfile(profile, start),
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
	profile, err := s.latestProfile(ctx)
	if err != nil {
		return ProfileResponse{
			GeneratedAt: time.Now().UnixMilli(),
			PatientName: "GlycoView Patient",
			Headline:    "No therapy profile is available yet",
		}, nil
	}
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
			{Title: "Clinical note", Detail: "This view is intended for review. Editing workflows should remain explicit and auditable.", Kind: "note", Accent: "violet"},
		},
	}, nil
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
