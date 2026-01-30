package entities

import "github.com/uptrace/bun"

type DocumentBase struct {
	// DB-friendly column names (snake_case) with JSON camelCase for API
	Identifier  string        `bun:"identifier" json:"identifier,omitempty"`
	Date        int64         `bun:"date,notnull" json:"date" validate:"required,gte=0"`
	UTCOffset   *int          `bun:"utc_offset" json:"utcOffset,omitempty"`
	App         string        `bun:"app,notnull" json:"app,omitempty" validate:"required"`
	Device      string        `bun:"device" json:"device,omitempty"`
	ID          string        `bun:"id,pk" json:"_id,omitempty"`
	SRVCreated  *int64        `bun:"srv_created" json:"srvCreated,omitempty"`
	Subject     string        `bun:"subject" json:"subject,omitempty"`
	SRVModified *int64        `bun:"srv_modified" json:"srvModified,omitempty"`
	ModifiedBy  string        `bun:"modified_by" json:"modifiedBy,omitempty"`
	IsValid     *bool         `bun:"is_valid" json:"isValid,omitempty"`
	IsReadOnly  *bool         `bun:"is_read_only" json:"isReadOnly,omitempty"`
	_bun        bun.BaseModel `bun:"-"`
}
