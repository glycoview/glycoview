package model

import (
	"github.com/go-playground/validator/v10"
	"github.com/uptrace/bun"
)

var validate = validator.New()

type Entry struct {
	bun.BaseModel `bun:"table:entries"`
	ID            string `bun:"id,pk,notnull" validate:"required,uuid4" json:"id"`
	Type          string `bun:"type,notnull" validate:"required,oneof=sgv mbg cal etc" json:"type"`
	DateString    string `bun:"date_string,notnull" validate:"required,datetime=2006-01-02T15:04:05Z07:00" json:"dateString"`
	Date          int64  `bun:"date,notnull" validate:"required,gte=0" json:"date"`
	SGV           int    `bun:"sgv,notnull" validate:"required,gte=0,lte=1000" json:"sgv"`
	Direction     string `bun:"direction,notnull" validate:"required,oneof=DoubleUp SingleUp FortyFiveUp Flat FortyFiveDown SingleDown DoubleDown NotComputable RateOutOfRange" json:"direction"`
	Noise         int    `bun:"noise,notnull" validate:"required,gte=0,lte=100" json:"noise"`
	Filtered      int    `bun:"filtered,notnull" validate:"required,gte=0,lte=1000" json:"filtered"`
	Unfiltered    int    `bun:"unfiltered,notnull" validate:"required,gte=0,lte=1000" json:"unfiltered"`
	RSSI          int    `bun:"rssi,notnull" validate:"required,gte=0,lte=100" json:"rssi"`
}

func (e *Entry) Validate() error {
	return validate.Struct(e)
}
