package presenter

import "github.com/better-monitoring/bscout/pkg/entities"

type Settings struct {
	DocumentBase
	// Placeholder for settings-specific fields
}

func SettingsSuccessResponse(items *[]entities.Settings) *[]Settings {
	var resp []Settings
	for _, v := range *items {
		s := Settings{DocumentBase: DocumentBaseFromEntity(v.DocumentBase)}
		resp = append(resp, s)
	}
	return &resp
}
