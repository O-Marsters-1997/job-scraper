package dto

import "time"

type TaskSort string

const (
	TaskSortCreated   TaskSort = "created"
	TaskSortJobPosted TaskSort = "job_posted"
)

type TaskType struct {
	ID        string
	Name      string
	Colour    string
	TaskCount int
	CreatedAt time.Time
}

type TaskTypeInput struct {
	ID     string `json:"-" path:"id"`
	Name   string `json:"name"`
	Colour string `json:"colour"`
}

type Task struct {
	ID             string
	TypeID         string
	TypeName       string
	TypeColour     string
	JobID          *string
	JobTitle       string
	JobCompanySlug string
	JobPostedAt    *time.Time
	Link           string
	Notes          string
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TaskPage struct {
	Items []Task `json:"items"`
	Total int    `json:"total"`
}

type TasksQuery struct {
	TypeID   string `json:"type_id"`
	JobID    string `json:"job_id"`
	Done     string `json:"done"`
	Sort     string `json:"sort"`
	Page     string `json:"page"`
	PageSize string `json:"page_size"`
}

type CreateTaskInput struct {
	TypeID string  `json:"type_id"`
	JobID  *string `json:"job_id"`
	Link   string  `json:"link"`
	Notes  string  `json:"notes"`
}

type UpdateTaskInput struct {
	ID     string `json:"-" path:"id"`
	TypeID string `json:"type_id"`
	Link   string `json:"link"`
	Notes  string `json:"notes"`
}

type TaskDoneInput struct {
	ID   string `json:"-" path:"id"`
	Done bool   `json:"done"`
}

type TaskPageOptions struct {
	TypeID string
	JobID  string
	Done   bool
	Sort   TaskSort
	Limit  int32
	Offset int32
}
