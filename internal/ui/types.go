package ui

type Metric struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Detail  string `json:"detail,omitempty"`
	Accent  string `json:"accent,omitempty"`
	Delta   string `json:"delta,omitempty"`
	Warning bool   `json:"warning,omitempty"`
}

type TimeInRangeBand struct {
	Label   string  `json:"label"`
	Range   string  `json:"range"`
	Minutes int     `json:"minutes"`
	Percent float64 `json:"percent"`
	Accent  string  `json:"accent"`
}

type GlucosePoint struct {
	At        int64   `json:"at"`
	Value     float64 `json:"value"`
	Direction string  `json:"direction,omitempty"`
}

type EventPoint struct {
	At       int64   `json:"at"`
	Label    string  `json:"label"`
	Kind     string  `json:"kind"`
	Value    float64 `json:"value"`
	Duration int     `json:"duration,omitempty"`
	Subtitle string  `json:"subtitle,omitempty"`
}

type DeviceCard struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Status     string   `json:"status"`
	LastSeen   int64    `json:"lastSeen"`
	Battery    string   `json:"battery,omitempty"`
	Reservoir  string   `json:"reservoir,omitempty"`
	Badge      string   `json:"badge,omitempty"`
	Connection string   `json:"connection,omitempty"`
	Details    []Metric `json:"details,omitempty"`
}

type ActivityItem struct {
	At      int64  `json:"at"`
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Kind    string `json:"kind"`
	Accent  string `json:"accent,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type OverviewResponse struct {
	GeneratedAt int64             `json:"generatedAt"`
	PatientName string            `json:"patientName"`
	Subtitle    string            `json:"subtitle"`
	Current     Metric            `json:"current"`
	Sparkline   []GlucosePoint    `json:"sparkline"`
	TimeInRange []TimeInRangeBand `json:"timeInRange"`
	NarrowRange TimeInRangeBand   `json:"narrowRange"`
	Metrics     []Metric          `json:"metrics"`
	Devices     []DeviceCard      `json:"devices"`
	Activity    []ActivityItem    `json:"activity"`
}

type DailyResponse struct {
	GeneratedAt  int64             `json:"generatedAt"`
	PatientName  string            `json:"patientName"`
	DateLabel    string            `json:"dateLabel"`
	RangeStart   int64             `json:"rangeStart"`
	RangeEnd     int64             `json:"rangeEnd"`
	Glucose      []GlucosePoint    `json:"glucose"`
	Carbs        []EventPoint      `json:"carbs"`
	Insulin      []EventPoint      `json:"insulin"`
	Boluses      []EventPoint      `json:"boluses"`
	SMBs         []EventPoint      `json:"smbs"`
	TempBasals   []EventPoint      `json:"tempBasals"`
	SMBGs        []EventPoint      `json:"smbgs"`
	BasalProfile []EventPoint      `json:"basalProfile"`
	TimeInRange  []TimeInRangeBand `json:"timeInRange"`
	Metrics      []Metric          `json:"metrics"`
	Devices      []DeviceCard      `json:"devices"`
}

type TrendBucket struct {
	Hour   int     `json:"hour"`
	P10    float64 `json:"p10"`
	P25    float64 `json:"p25"`
	P50    float64 `json:"p50"`
	P75    float64 `json:"p75"`
	P90    float64 `json:"p90"`
	Points int     `json:"points"`
}

type DailySummary struct {
	Day        string  `json:"day"`
	Date       int64   `json:"date"`
	AvgGlucose float64 `json:"avgGlucose"`
	Carbs      float64 `json:"carbs"`
	Insulin    float64 `json:"insulin"`
	TIR        float64 `json:"tir"`
}

type TrendsResponse struct {
	GeneratedAt int64             `json:"generatedAt"`
	PatientName string            `json:"patientName"`
	RangeLabel  string            `json:"rangeLabel"`
	Days        int               `json:"days"`
	AGP         []TrendBucket     `json:"agp"`
	TimeInRange []TimeInRangeBand `json:"timeInRange"`
	Metrics     []Metric          `json:"metrics"`
	DaysSummary []DailySummary    `json:"daysSummary"`
}

type SchedulePoint struct {
	Time  string `json:"time"`
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

type ProfileResponse struct {
	GeneratedAt   int64           `json:"generatedAt"`
	PatientName   string          `json:"patientName"`
	Headline      string          `json:"headline"`
	Metrics       []Metric        `json:"metrics"`
	BasalSchedule []SchedulePoint `json:"basalSchedule"`
	CarbRatios    []SchedulePoint `json:"carbRatios"`
	Sensitivity   []SchedulePoint `json:"sensitivity"`
	Targets       []SchedulePoint `json:"targets"`
	Notes         []ActivityItem  `json:"notes"`
}

type DevicesResponse struct {
	GeneratedAt int64          `json:"generatedAt"`
	PatientName string         `json:"patientName"`
	Headline    string         `json:"headline"`
	Cards       []DeviceCard   `json:"cards"`
	Metrics     []Metric       `json:"metrics"`
	Activity    []ActivityItem `json:"activity"`
}
