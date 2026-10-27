package dto

import "time"

type Notification struct {
	ID        string     `json:"id"`
	JobID     string     `json:"job_id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}
