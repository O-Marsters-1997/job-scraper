package dto

type ScoringOptionsView struct {
	Dimensions []DimensionSpec `json:"dimensions"`
	Options    []ScoringOption `json:"options"`
	Custom     []CustomOption  `json:"custom"`
	CustomCap  int             `json:"customCap"`
}
