package dto

import "time"

type ScoringEffect struct {
	ID             string
	JobID          string
	UserID         string
	Fingerprint    string
	ConfigVersion  time.Time
	Model          string
	Attempts       int
	FirstDiscovery bool
}

type ScoringFailure struct {
	Reason     string
	Terminal   bool
	RetryAfter time.Duration
}
