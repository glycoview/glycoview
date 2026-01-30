package presenter

import "github.com/better-monitoring/bscout/pkg/entities"

type DocumentBase struct {
	Identifier  string `json:"identifier,omitempty"`
	Date        int64  `json:"date"`
	UTCOffset   *int   `json:"utcOffset,omitempty"`
	App         string `json:"app,omitempty"`
	Device      string `json:"device,omitempty"`
	ID          string `json:"_id,omitempty"`
	SRVCreated  *int64 `json:"srvCreated,omitempty"`
	Subject     string `json:"subject,omitempty"`
	SRVModified *int64 `json:"srvModified,omitempty"`
	ModifiedBy  string `json:"modifiedBy,omitempty"`
	IsValid     *bool  `json:"isValid,omitempty"`
	IsReadOnly  *bool  `json:"isReadOnly,omitempty"`
}

func DocumentBaseFromEntity(d entities.DocumentBase) DocumentBase {
	return DocumentBase{
		Identifier:  d.Identifier,
		Date:        d.Date,
		UTCOffset:   d.UTCOffset,
		App:         d.App,
		Device:      d.Device,
		ID:          d.ID,
		SRVCreated:  d.SRVCreated,
		Subject:     d.Subject,
		SRVModified: d.SRVModified,
		ModifiedBy:  d.ModifiedBy,
		IsValid:     d.IsValid,
		IsReadOnly:  d.IsReadOnly,
	}
}
