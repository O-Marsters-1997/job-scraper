package dto

type DimensionSpec struct {
	Key     Dimension `json:"key"`
	Kind    string    `json:"kind"`
	Stances []string  `json:"stances"`
}
