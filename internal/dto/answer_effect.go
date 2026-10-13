package dto

import "time"

type AnswerEffect struct {
	ID             string
	JobID          string
	UserID         string
	Fingerprint    string
	Model          string
	Attempts       int
	FirstDiscovery bool
}

type ScoringFailure struct {
	Reason     string
	Terminal   bool
	RetryAfter time.Duration
}
