package dto

import "time"

type TrackedBoard struct {
	ID         string      `json:"id"`
	Source     string      `json:"source"`
	BoardToken string      `json:"board_token"`
	Status     BoardStatus `json:"status"`
	URL        string      `json:"url"`
}

type TrackedCompany struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Slug                 string         `json:"slug"`
	Enabled              bool           `json:"enabled"`
	ReviewState          string         `json:"review_state"`
	CheckIntervalMinutes int            `json:"check_interval_minutes"`
	Boards               []TrackedBoard `json:"boards"`
	OpenJobs             int            `json:"open_jobs"`
	RelevantJobs         int            `json:"relevant_jobs"`
	LastCheckedAt        *time.Time     `json:"last_checked_at"`
}
