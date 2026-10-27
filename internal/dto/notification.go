package dto

import "time"

type Notification struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	JobID     string          `json:"job_id"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Job       NotificationJob `json:"job"`
	CreatedAt time.Time       `json:"created_at"`
	ReadAt    *time.Time      `json:"read_at"`
}

type NotificationJob struct {
	Title       string `json:"title"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	SalaryRaw   string `json:"salary_raw"`
	Band        string `json:"band"`
	Score       *int   `json:"score"`
}
