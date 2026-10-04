package store

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store/sqlc"
)

func fromOptionalDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func toApplicationDTO(a sqlc.Application) dto.Application {
	return dto.Application{
		ID:         a.ID.String(),
		UserID:     a.UserID.String(),
		JobID:      a.JobID.String(),
		StatusID:   a.StatusID.String(),
		Notes:      a.Notes.String,
		AppliedAt:  fromOptionalDate(a.AppliedAt),
		SalaryInfo: a.SalaryInfo.String,
		CreatedAt:  a.CreatedAt.Time,
		UpdatedAt:  a.UpdatedAt.Time,
		ChaseBy:    fromOptionalDate(a.ChaseBy),
	}
}

func toApplicationWithDetailsDTO(r sqlc.ListApplicationsRow) dto.ApplicationWithDetails {
	return dto.ApplicationWithDetails{
		ID:             r.ID.String(),
		UserID:         r.UserID.String(),
		JobID:          r.JobID.String(),
		JobTitle:       r.JobTitle,
		JobCompanySlug: r.JobCompanySlug,
		JobLocation:    r.JobLocation,
		JobURL:         r.JobUrl,
		StatusID:       r.StatusID.String(),
		StatusName:     r.StatusName.String,
		StatusColour:   r.StatusColour.String,
		Notes:          r.Notes.String,
		AppliedAt:      fromOptionalDate(r.AppliedAt),
		SalaryInfo:     r.SalaryInfo.String,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
		ChaseBy:        fromOptionalDate(r.ChaseBy),
	}
}

func toApplicationStatusDTO(s sqlc.ApplicationStatus) dto.ApplicationStatus {
	return dto.ApplicationStatus{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		Name:      s.Name,
		Colour:    s.Colour,
		CreatedAt: s.CreatedAt.Time,
	}
}

func toJobApplicationSummaryDTO(r sqlc.GetApplicationsForJobsRow) dto.JobApplicationSummary {
	return dto.JobApplicationSummary{
		ApplicationID: r.ID.String(),
		StatusID:      r.StatusID.String(),
		StatusName:    r.StatusName.String,
		StatusColour:  r.StatusColour.String,
	}
}
