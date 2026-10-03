package dto

import "time"

type JobPageOptions struct {
	Limit        int32
	CursorTime   time.Time
	CursorID     string
	Availability string
	CompanyID    string
	ScoredOnly   bool
	SinceDays    int32

	ExcludedCompanySlugs []string
}

type JobPage struct {
	Items      []Job  `json:"items"`
	NextCursor string `json:"next_cursor"`
}
