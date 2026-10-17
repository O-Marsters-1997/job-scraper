package store

import (
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store/sqlc"
)

func toTaskTypeDTO(t sqlc.ListTaskTypesRow) dto.TaskType {
	return dto.TaskType{
		ID:        t.ID.String(),
		Name:      t.Name,
		Colour:    t.Colour,
		TaskCount: int(t.TaskCount),
		CreatedAt: t.CreatedAt.Time,
	}
}

func toTaskDTO(r sqlc.PageTasksRow) dto.Task {
	var jobID *string
	if r.JobID.Valid {
		jobID = new(r.JobID.String())
	}
	return dto.Task{
		ID:             r.ID.String(),
		TypeID:         r.TypeID.String(),
		TypeName:       r.TypeName,
		TypeColour:     r.TypeColour,
		JobID:          jobID,
		JobTitle:       r.JobTitle.String,
		JobCompanySlug: r.JobCompanySlug.String,
		JobPostedAt:    data.TimePtr(r.JobPostedAt),
		Link:           r.Link.String,
		Notes:          r.Notes.String,
		CompletedAt:    data.TimePtr(r.CompletedAt),
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}
}
