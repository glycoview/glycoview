package entities

import (
	"github.com/uptrace/bun"
)

type Entry struct {
	bun.BaseModel `bun:"table:entries"`
	DocumentBase
	Type       string `json:"type" validate:"required,oneof=sgv mbg cal etc"`
	SGV        *int   `json:"sgv,omitempty" validate:"omitempty,gte=0,lte=1000"`
	Direction  string `json:"direction,omitempty"`
	Noise      *int   `json:"noise,omitempty" validate:"omitempty,gte=0,lte=100"`
	Filtered   *int   `json:"filtered,omitempty" validate:"omitempty,gte=0,lte=1000"`
	Unfiltered *int   `json:"unfiltered,omitempty" validate:"omitempty,gte=0,lte=1000"`
	RSSI       *int   `json:"rssi,omitempty" validate:"omitempty,gte=0,lte=100"`
	Units      string `json:"units,omitempty"`
	DateString string `json:"dateString,omitempty"`
}

func (e *Entry) Validate() error {
	return validate.Struct(e)
}
