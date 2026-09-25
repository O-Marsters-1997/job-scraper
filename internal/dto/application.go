package dto

import (
	"time"

	"github.com/ollymarsters/job-scraper/internal/fp"
)

type CreateApplicationInput struct {
	JobID      string            `json:"job_id"`
	StatusID   string            `json:"status_id"`
	Notes      string            `json:"notes"`
	SalaryInfo string            `json:"salary_info"`
	AppliedAt  fp.Option[string] `json:"applied_at"`
}

type UpdateApplicationInput struct {
	ID         string            `json:"-" path:"id"`
	StatusID   string            `json:"status_id"`
	Notes      string            `json:"notes"`
	SalaryInfo string            `json:"salary_info"`
	AppliedAt  fp.Option[string] `json:"applied_at"`
}

// ApplicationsQuery is the query-string shape for GET /applications.
type ApplicationsQuery struct {
	StatusID string `json:"status_id"`
}

// ApplicationsForJobsQuery is the query-string shape for GET
// /applications/for-jobs: JobIDs is a comma-separated list of job IDs.
type ApplicationsForJobsQuery struct {
	JobIDs string `json:"job_ids"`
}

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
