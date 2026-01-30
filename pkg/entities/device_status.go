package entities

// DeviceStatus represents state of a physical device; extends DocumentBase
type DeviceStatus struct {
	DocumentBase
	// Additional device-specific properties can be added here.
	SomeProperty string `json:"some_property,omitempty"`
}
