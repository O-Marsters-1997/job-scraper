package dto

import "time"

type BoardPoll struct {
	ID              string
	CompanyID       string
	CompanySlug     string
	Source          string
	Token           string
	IntervalMinutes int
	LeaseOwner      string
	Version         int64
	StartedAt       time.Time
	Manual          bool
}

type BoardSnapshot struct {
	Poll     BoardPoll
	Jobs     []Job
	Complete bool
	// NextPollIn, when positive, replaces the default interval before the next poll.
	NextPollIn time.Duration
}
