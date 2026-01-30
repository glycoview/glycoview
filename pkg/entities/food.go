package entities

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
