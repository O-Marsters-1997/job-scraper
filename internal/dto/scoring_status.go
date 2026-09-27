package dto

type ScoringStatus struct {
	Pending int64 `json:"pending"`
}

type RecomputeResult struct {
	Recomputed int64 `json:"recomputed"`
}
