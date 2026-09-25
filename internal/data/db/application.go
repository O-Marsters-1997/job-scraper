package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromApplication(a pgsqlc.Application) dto.Application {
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
	}
}

func fromApplicationListRow(r pgsqlc.ListApplicationsByUserRow) dto.ApplicationWithDetails {
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
	}
}

func fromApplicationListStatusRow(r pgsqlc.ListApplicationsByUserAndStatusRow) dto.ApplicationWithDetails {
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
	}
}

func (db *DB) CreateApplication(ctx context.Context, input dto.CreateApplicationInput) (dto.Application, error) {
	uid, err := parseUUID(input.UserID)
	if err != nil {
		return dto.Application{}, err
	}
	jid, err := parseUUID(input.JobID)
	if err != nil {
		return dto.Application{}, err
	}
	var sid pgtype.UUID
	if input.StatusID != "" {
		sid, err = parseUUID(input.StatusID)
		if err != nil {
			return dto.Application{}, err
		}
	}
	appliedAt, err := toOptionalDate(input.AppliedAt)
	if err != nil {
		return dto.Application{}, err
	}
	a, err := db.queries.CreateApplication(ctx, pgsqlc.CreateApplicationParams{
		UserID:     uid,
		JobID:      jid,
		StatusID:   sid,
		Notes:      pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		AppliedAt:  appliedAt,
		SalaryInfo: pgtype.Text{String: input.SalaryInfo, Valid: input.SalaryInfo != ""},
	})
	if err != nil {
		return dto.Application{}, fmt.Errorf("db.CreateApplication: %w", err)
	}
	return fromApplication(a), nil
}

func (db *DB) ListApplicationsByUser(ctx context.Context, userID string) ([]dto.ApplicationWithDetails, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListApplicationsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListApplicationsByUser: %w", err)
	}
	out := make([]dto.ApplicationWithDetails, len(rows))
	for i, row := range rows {
		out[i] = fromApplicationListRow(row)
	}
	return out, nil
}

func (db *DB) ListApplicationsByUserAndStatus(ctx context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	sid, err := parseUUID(statusID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListApplicationsByUserAndStatus(ctx, pgsqlc.ListApplicationsByUserAndStatusParams{
		UserID:   uid,
		StatusID: sid,
	})
	if err != nil {
		return nil, fmt.Errorf("db.ListApplicationsByUserAndStatus: %w", err)
	}
	out := make([]dto.ApplicationWithDetails, len(rows))
	for i, row := range rows {
		out[i] = fromApplicationListStatusRow(row)
	}
	return out, nil
}

func (db *DB) UpdateApplication(ctx context.Context, input dto.UpdateApplicationInput) (dto.Application, error) {
	aid, err := parseUUID(input.ID)
	if err != nil {
		return dto.Application{}, err
	}
	uid, err := parseUUID(input.UserID)
	if err != nil {
		return dto.Application{}, err
	}
	var sid pgtype.UUID
	if input.StatusID != "" {
		sid, err = parseUUID(input.StatusID)
		if err != nil {
			return dto.Application{}, err
		}
	}
	appliedAt, err := toOptionalDate(input.AppliedAt)
	if err != nil {
		return dto.Application{}, err
	}
	a, err := db.queries.UpdateApplication(ctx, pgsqlc.UpdateApplicationParams{
		ID:         aid,
		UserID:     uid,
		StatusID:   sid,
		Notes:      pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		AppliedAt:  appliedAt,
		SalaryInfo: pgtype.Text{String: input.SalaryInfo, Valid: input.SalaryInfo != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.Application{}, providers.ErrNotFound
		}
		return dto.Application{}, fmt.Errorf("db.UpdateApplication: %w", err)
	}
	return fromApplication(a), nil
}

func (db *DB) DeleteApplication(ctx context.Context, id, userID string) error {
	aid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.DeleteApplication(ctx, pgsqlc.DeleteApplicationParams{
		ID:     aid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("db.DeleteApplication: %w", err)
	}
	return nil
}

func (db *DB) GetApplicationsForJobs(ctx context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	pgIDs := make([]pgtype.UUID, 0, len(jobIDs))
	for _, id := range jobIDs {
		jid, err := parseUUID(id)
		if err != nil {
			return nil, err
		}
		pgIDs = append(pgIDs, jid)
	}
	rows, err := db.queries.GetApplicationsForJobs(ctx, pgsqlc.GetApplicationsForJobsParams{
		UserID:  uid,
		Column2: pgIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("db.GetApplicationsForJobs: %w", err)
	}
	out := make(map[string]dto.JobApplicationSummary, len(rows))
	for _, row := range rows {
		out[row.JobID.String()] = dto.JobApplicationSummary{
			ApplicationID: row.ID.String(),
			StatusID:      row.StatusID.String(),
			StatusName:    row.StatusName.String,
			StatusColour:  row.StatusColour.String,
		}
	}
	return out, nil
}
