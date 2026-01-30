package presenter

import "github.com/better-monitoring/bscout/pkg/entities"

type Food struct {
	DocumentBase
	Food         string   `json:"food,omitempty"`
	Category     string   `json:"category,omitempty"`
	Subcategory  string   `json:"subcategory,omitempty"`
	Name         string   `json:"name,omitempty"`
	Portion      *float64 `json:"portion,omitempty"`
	Unit         string   `json:"unit,omitempty"`
	Carbs        *float64 `json:"carbs,omitempty"`
	Fat          *float64 `json:"fat,omitempty"`
	Protein      *float64 `json:"protein,omitempty"`
	Energy       *float64 `json:"energy,omitempty"`
	GI           *float64 `json:"gi,omitempty"`
	HideAfterUse *bool    `json:"hideafteruse,omitempty"`
	Hidden       *bool    `json:"hidden,omitempty"`
	Position     *int     `json:"position,omitempty"`
	Portions     *float64 `json:"portions,omitempty"`
	Foods        []Food   `json:"foods,omitempty"`
}

func FoodSuccessResponse(items *[]entities.Food) *[]Food {
	var resp []Food
	for _, v := range *items {
		f := Food{
			DocumentBase: DocumentBaseFromEntity(v.DocumentBase),
			Food:         v.Food,
			Category:     v.Category,
			Subcategory:  v.Subcategory,
			Name:         v.Name,
			Portion:      v.Portion,
			Unit:         v.Unit,
			Carbs:        v.Carbs,
			Fat:          v.Fat,
			Protein:      v.Protein,
			Energy:       v.Energy,
			GI:           v.GI,
			HideAfterUse: v.HideAfterUse,
			Hidden:       v.Hidden,
			Position:     v.Position,
			Portions:     v.Portions,
			Foods:        nil,
		}
		// Note: nested Foods conversion omitted to avoid recursive deep copy cycles.
		resp = append(resp, f)
	}
	return &resp
}
