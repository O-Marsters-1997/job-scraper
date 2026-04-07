package dto

import "time"

type ApplicationStatus struct {
	ID        string
	UserID    string
	Name      string
	Colour    string
	CreatedAt time.Time
}

type Application struct {
	ID         string
	UserID     string
	JobID      string
	StatusID   string
	Notes      string
	AppliedAt  *time.Time
	SalaryInfo string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ApplicationWithDetails struct {
	ID             string
	UserID         string
	JobID          string
	JobTitle       string
	JobCompanySlug string
	JobLocation    string
	JobURL         string
	StatusID       string
	StatusName     string
	StatusColour   string
	Notes          string
	AppliedAt      *time.Time
	SalaryInfo     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type JobApplicationSummary struct {
	ApplicationID string
	StatusID      string
	StatusName    string
	StatusColour  string
}
