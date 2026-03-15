package dto

import "time"

type Job struct {
	Title       string
	Location    string
	URL         string
	CompanySlug string
	Source      string
	UpdatedAt   time.Time
}
