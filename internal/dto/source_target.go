package dto

import "time"

type SourceTarget struct {
	ID                   string
	UserID               string
	Source               string
	Value                string
	URL                  string
	Enabled              bool
	Filters              map[string]string
	CompanyID            string
	CheckIntervalMinutes int
	LastCheckedAt        *time.Time
	RunStatus            string
	RunID                string
	LastRunAt            *time.Time
	LastSucceededAt      *time.Time
	LastRunError         string
	DisabledReason       string
	UpdatedAt            time.Time
}
