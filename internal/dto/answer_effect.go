package dto

import "time"

// AnswerEffect is one queued job that needs its bank questions answered and
// scored, one row per job rather than per (job, user).
type AnswerEffect struct {
	ID             string
	JobID          string
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
