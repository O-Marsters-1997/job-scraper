package dto

type DimensionSpec struct {
	Key        Dimension `json:"key"`
	Kind       string    `json:"kind"`
	Stances    []Stance  `json:"stances"`
	Weight     float64   `json:"weight"`
	Saturation int       `json:"saturation"`
}
