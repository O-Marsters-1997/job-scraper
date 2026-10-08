package dto

import (
	"time"
)

type CreateApplicationInput struct {
	JobID      string  `json:"job_id"`
	StatusID   string  `json:"status_id"`
	Notes      string  `json:"notes"`
	SalaryInfo string  `json:"salary_info"`
	AppliedAt  *string `json:"applied_at"`
}

type UpdateApplicationInput struct {
	ID         string  `json:"-" path:"id"`
	StatusID   string  `json:"status_id"`
	Notes      string  `json:"notes"`
	SalaryInfo string  `json:"salary_info"`
	AppliedAt  *string `json:"applied_at"`
}

type ApplicationsQuery struct {
	StatusID string `json:"status_id"`
	Chase    bool   `json:"chase,string"`
}

type ChaseInput struct {
	ID      string `json:"-" path:"id"`
	ChaseBy string `json:"chase_by"`
}

type ApplicationsForJobsQuery struct {
	JobIDs string `json:"job_ids"`
}

type ApplicationStatus struct {
	ID              string
	UserID          string
	Name            string
	Colour          string
	ReplyWindowDays *int
	CreatedAt       time.Time
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
	ChaseBy    *time.Time
}

type ApplicationWithDetails struct {
	ID             string
	UserID         string
	JobID          string
	JobTitle       string
	JobCompanySlug string
	JobLocation    string
	JobURL         string
	JobClosedAt    *time.Time
	StatusID       string
	StatusName     string
	StatusColour   string
	Notes          string
	AppliedAt      *time.Time
	SalaryInfo     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ChaseBy        *time.Time
}

type JobApplicationSummary struct {
	ApplicationID string
	StatusID      string
	StatusName    string
	StatusColour  string
}
