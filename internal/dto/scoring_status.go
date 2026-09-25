package dto

type ScoringStatus struct {
	Pending int64 `json:"pending"`
	Failed  int64 `json:"failed"`
	Stale   int64 `json:"stale"`
}

type RescoreResult struct {
	Queued int64 `json:"queued"`
}
