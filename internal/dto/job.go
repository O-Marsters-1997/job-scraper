package dto

import "time"

type Job struct {
	ID          string
	Title       string
	Location    string
	URL         string
	CompanySlug string
	Source      string
	UpdatedAt   time.Time
	ScrapedAt   time.Time
	Description string
	SalaryRaw   string
}
