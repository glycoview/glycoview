package entities

type Treatment struct {
	DocumentBase
	EventType    string   `json:"eventType,omitempty"`
	Glucose      string   `json:"glucose,omitempty"`
	GlucoseType  string   `json:"glucoseType,omitempty"`
	Units        string   `json:"units,omitempty"`
	Carbs        *float64 `json:"carbs,omitempty"`
	Protein      *float64 `json:"protein,omitempty"`
	Fat          *float64 `json:"fat,omitempty"`
	Insulin      *float64 `json:"insulin,omitempty"`
	Duration     *float64 `json:"duration,omitempty"`
	PreBolus     *float64 `json:"preBolus,omitempty"`
	SplitNow     *float64 `json:"splitNow,omitempty"`
	SplitExt     *float64 `json:"splitExt,omitempty"`
	Percent      *float64 `json:"percent,omitempty"`
	Absolute     *float64 `json:"absolute,omitempty"`
	TargetTop    *float64 `json:"targetTop,omitempty"`
	TargetBottom *float64 `json:"targetBottom,omitempty"`
	ProfileName  string   `json:"profile,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	EnteredBy    string   `json:"enteredBy,omitempty"`
}
