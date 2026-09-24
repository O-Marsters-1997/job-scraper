package dto

import "time"

type SourceTarget struct {
	ID                   string
	UserID               string
	Source               string
	Value                string
	Enabled              bool
	Filters              map[string]string
	CompanyID            string
	CheckIntervalMinutes int
	LastCheckedAt        *time.Time
	RunStatus            string
	RunID                string
	LastRunAt            *time.Time
	LastRunError         string
	UpdatedAt            time.Time
}
