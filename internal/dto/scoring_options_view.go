package dto

type ScoringOptionsView struct {
	Dimensions []DimensionSpec `json:"dimensions"`
	Options    []ScoringOption `json:"options"`
}
