package presenter

import "github.com/better-monitoring/bscout/pkg/entities"

type Profile struct {
	DocumentBase
	// Placeholder for profile specific fields
}

func ProfileSuccessResponse(items *[]entities.Profile) *[]Profile {
	var resp []Profile
	for _, v := range *items {
		p := Profile{DocumentBase: DocumentBaseFromEntity(v.DocumentBase)}
		resp = append(resp, p)
	}
	return &resp
}
