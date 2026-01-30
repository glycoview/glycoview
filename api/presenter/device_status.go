package presenter

import "github.com/better-monitoring/bscout/pkg/entities"

type DeviceStatus struct {
	DocumentBase
	SomeProperty string `json:"some_property,omitempty"`
}

func DeviceStatusSuccessResponse(items *[]entities.DeviceStatus) *[]DeviceStatus {
	var resp []DeviceStatus
	for _, v := range *items {
		d := DeviceStatus{
			DocumentBase: DocumentBaseFromEntity(v.DocumentBase),
			SomeProperty: v.SomeProperty,
		}
		resp = append(resp, d)
	}
	return &resp
}
