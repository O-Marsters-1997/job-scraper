package dto

type DimensionSpec struct {
	Key        Dimension `json:"key"`
	Kind       string    `json:"kind"`
	Stances    []string  `json:"stances"`
	Gate       bool      `json:"gate"`
	Weight     float64   `json:"weight"`
	Saturation int       `json:"saturation"`
}
