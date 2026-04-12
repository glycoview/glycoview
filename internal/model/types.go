package model

type Entry struct {
	Identifier string         `json:"identifier,omitempty"`
	Type       string         `json:"type,omitempty"`
	SGV        any            `json:"sgv,omitempty"`
	MBG        any            `json:"mbg,omitempty"`
	Date       int64          `json:"date,omitempty"`
	DateString string         `json:"dateString,omitempty"`
	UTCOffset  int            `json:"utcOffset,omitempty"`
	Device     string         `json:"device,omitempty"`
	Direction  string         `json:"direction,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type Treatment struct {
	Identifier  string         `json:"identifier,omitempty"`
	EventType   string         `json:"eventType,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	Date        int64          `json:"date,omitempty"`
	UTCOffset   int            `json:"utcOffset,omitempty"`
	Insulin     any            `json:"insulin,omitempty"`
	Carbs       any            `json:"carbs,omitempty"`
	Glucose     any            `json:"glucose,omitempty"`
	GlucoseType string         `json:"glucoseType,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	EnteredBy   string         `json:"enteredBy,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

type DeviceStatus struct {
	Identifier string         `json:"identifier,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
	Date       int64          `json:"date,omitempty"`
	UTCOffset  int            `json:"utcOffset,omitempty"`
	Device     string         `json:"device,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type Profile struct {
	Identifier     string         `json:"identifier,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
	DefaultProfile string         `json:"defaultProfile,omitempty"`
	StartDate      string         `json:"startDate,omitempty"`
	Store          map[string]any `json:"store,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
}

type Food struct {
	Identifier string         `json:"identifier,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
	Name       string         `json:"name,omitempty"`
	Category   string         `json:"category,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type Setting struct {
	Identifier string         `json:"identifier,omitempty"`
	Key        string         `json:"key,omitempty"`
	Value      any            `json:"value,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}
