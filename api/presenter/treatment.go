package presenter

import "github.com/better-monitoring/bscout/pkg/entities"

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

func TreatmentSuccessResponse(items *[]entities.Treatment) *[]Treatment {
	var resp []Treatment
	for _, v := range *items {
		t := Treatment{
			DocumentBase: DocumentBaseFromEntity(v.DocumentBase),
			EventType:    v.EventType,
			Glucose:      v.Glucose,
			GlucoseType:  v.GlucoseType,
			Units:        v.Units,
			Carbs:        v.Carbs,
			Protein:      v.Protein,
			Fat:          v.Fat,
			Insulin:      v.Insulin,
			Duration:     v.Duration,
			PreBolus:     v.PreBolus,
			SplitNow:     v.SplitNow,
			SplitExt:     v.SplitExt,
			Percent:      v.Percent,
			Absolute:     v.Absolute,
			TargetTop:    v.TargetTop,
			TargetBottom: v.TargetBottom,
			ProfileName:  v.ProfileName,
			Reason:       v.Reason,
			Notes:        v.Notes,
			EnteredBy:    v.EnteredBy,
		}
		resp = append(resp, t)
	}
	return &resp
}
