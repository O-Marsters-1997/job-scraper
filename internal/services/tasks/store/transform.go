package store

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store/sqlc"
)

func toTaskTypeDTO(t sqlc.ListTaskTypesRow) dto.TaskType {
	return dto.TaskType{
		ID:        t.ID.String(),
		Name:      t.Name,
		Colour:    t.Colour,
		SystemKey: dto.TaskTypeSystemKey(t.SystemKey.String),
		TaskCount: int(t.TaskCount),
		CreatedAt: t.CreatedAt.Time,
	}
}

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	return new(u.String())
}

func toTaskDTO(r sqlc.PageTasksRow) dto.Task {
	return dto.Task{
		ID:             r.ID.String(),
		TypeID:         r.TypeID.String(),
		TypeName:       r.TypeName,
		TypeColour:     r.TypeColour,
		TypeSystemKey:  dto.TaskTypeSystemKey(r.TypeSystemKey.String),
		JobID:          uuidPtr(r.JobID),
		JobTitle:       r.JobTitle.String,
		JobCompanySlug: r.JobCompanySlug.String,
		JobCompanyID:   uuidPtr(r.JobCompanyID),
		ApplicationID:  uuidPtr(r.ApplicationID),
		JobPostedAt:    data.TimePtr(r.JobPostedAt),
		Link:           r.Link.String,
		Notes:          r.Notes.String,
		CompletedAt:    data.TimePtr(r.CompletedAt),
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}
}
