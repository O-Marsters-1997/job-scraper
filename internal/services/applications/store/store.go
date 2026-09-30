// Package store is the applications context's Postgres store: applications
// and application_statuses, plus a read-join onto jobs for the list view.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store/sqlc"
)

var ErrApplicationExists = apperr.Conflict("application already exists for this job")

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func parseOptionalDate(s *string) (pgtype.Date, error) {
	if s == nil {
		return pgtype.Date{}, nil
	}
	var d pgtype.Date
	if err := d.Scan(*s); err != nil {
		return pgtype.Date{}, fmt.Errorf("invalid applied date: %w", err)
	}
	return d, nil
}

func (s *Store) CreateApplication(ctx context.Context, userID string, input dto.CreateApplicationInput) (dto.Application, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Application{}, err
	}
	jid, err := data.UUID(input.JobID)
	if err != nil {
		return dto.Application{}, err
	}
	var sid pgtype.UUID
	if input.StatusID != "" {
		sid, err = data.UUID(input.StatusID)
		if err != nil {
			return dto.Application{}, err
		}
	}
	appliedAt, err := parseOptionalDate(input.AppliedAt)
	if err != nil {
		return dto.Application{}, err
	}
	a, err := s.queries.CreateApplication(ctx, sqlc.CreateApplicationParams{
		UserID:     uid,
		JobID:      jid,
		StatusID:   sid,
		Notes:      data.Text(input.Notes),
		AppliedAt:  appliedAt,
		SalaryInfo: data.Text(input.SalaryInfo),
	})
	if err != nil {
		if data.IsUniqueViolation(err) {
			return dto.Application{}, ErrApplicationExists
		}
		return dto.Application{}, fmt.Errorf("store.CreateApplication: %w", err)
	}
	return toApplicationDTO(a), nil
}

func (s *Store) ListApplicationsByUser(ctx context.Context, userID string) ([]dto.ApplicationWithDetails, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListApplicationsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListApplicationsByUser: %w", err)
	}
	out := make([]dto.ApplicationWithDetails, len(rows))
	for i, row := range rows {
		out[i] = toApplicationWithDetailsDTO(row)
	}
	return out, nil
}

func (s *Store) ListApplicationsByUserAndStatus(ctx context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	sid, err := data.UUID(statusID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListApplicationsByUserAndStatus(ctx, sqlc.ListApplicationsByUserAndStatusParams{
		UserID:   uid,
		StatusID: sid,
	})
	if err != nil {
		return nil, fmt.Errorf("store.ListApplicationsByUserAndStatus: %w", err)
	}
	out := make([]dto.ApplicationWithDetails, len(rows))
	for i, row := range rows {
		out[i] = toApplicationWithDetailsDTO(sqlc.ListApplicationsByUserRow(row))
	}
	return out, nil
}

func (s *Store) UpdateApplication(ctx context.Context, userID, id string, input dto.UpdateApplicationInput) (dto.Application, error) {
	aid, err := data.UUID(id)
	if err != nil {
		return dto.Application{}, err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Application{}, err
	}
	var sid pgtype.UUID
	if input.StatusID != "" {
		sid, err = data.UUID(input.StatusID)
		if err != nil {
			return dto.Application{}, err
		}
	}
	appliedAt, err := parseOptionalDate(input.AppliedAt)
	if err != nil {
		return dto.Application{}, err
	}
	a, err := s.queries.UpdateApplication(ctx, sqlc.UpdateApplicationParams{
		ID:         aid,
		UserID:     uid,
		StatusID:   sid,
		Notes:      data.Text(input.Notes),
		AppliedAt:  appliedAt,
		SalaryInfo: data.Text(input.SalaryInfo),
	})
	if err != nil {
		return dto.Application{}, data.QueryErr("UpdateApplication", err)
	}
	return toApplicationDTO(a), nil
}

func (s *Store) DeleteApplication(ctx context.Context, userID, id string) error {
	aid, err := data.UUID(id)
	if err != nil {
		return err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteApplication(ctx, sqlc.DeleteApplicationParams{
		ID:     aid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("store.DeleteApplication: %w", err)
	}
	return nil
}

func (s *Store) GetApplicationsForJobs(ctx context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	pgIDs := make([]pgtype.UUID, 0, len(jobIDs))
	for _, id := range jobIDs {
		jid, err := data.UUID(id)
		if err != nil {
			return nil, err
		}
		pgIDs = append(pgIDs, jid)
	}
	rows, err := s.queries.GetApplicationsForJobs(ctx, sqlc.GetApplicationsForJobsParams{
		UserID:  uid,
		Column2: pgIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("store.GetApplicationsForJobs: %w", err)
	}
	out := make(map[string]dto.JobApplicationSummary, len(rows))
	for _, row := range rows {
		out[row.JobID.String()] = toJobApplicationSummaryDTO(row)
	}
	return out, nil
}

func (s *Store) SeedDefaultStatuses(ctx context.Context, tx pgx.Tx, userID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.WithTx(tx).SeedDefaultStatuses(ctx, uid); err != nil {
		return fmt.Errorf("store.SeedDefaultStatuses: %w", err)
	}
	return nil
}

func (s *Store) CreateApplicationStatus(ctx context.Context, userID, name, colour string) (dto.ApplicationStatus, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	st, err := s.queries.CreateApplicationStatus(ctx, sqlc.CreateApplicationStatusParams{
		UserID: uid,
		Name:   name,
		Colour: colour,
	})
	if err != nil {
		return dto.ApplicationStatus{}, fmt.Errorf("store.CreateApplicationStatus: %w", err)
	}
	return toApplicationStatusDTO(st), nil
}

func (s *Store) ListApplicationStatusesByUser(ctx context.Context, userID string) ([]dto.ApplicationStatus, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListApplicationStatusesByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListApplicationStatusesByUser: %w", err)
	}
	out := make([]dto.ApplicationStatus, len(rows))
	for i, row := range rows {
		out[i] = toApplicationStatusDTO(row)
	}
	return out, nil
}

func (s *Store) UpdateApplicationStatus(ctx context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error) {
	sid, err := data.UUID(id)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	st, err := s.queries.UpdateApplicationStatus(ctx, sqlc.UpdateApplicationStatusParams{
		ID:     sid,
		UserID: uid,
		Name:   name,
		Colour: colour,
	})
	if err != nil {
		return dto.ApplicationStatus{}, fmt.Errorf("store.UpdateApplicationStatus: %w", err)
	}
	return toApplicationStatusDTO(st), nil
}

func (s *Store) DeleteApplicationStatus(ctx context.Context, id, userID string) error {
	sid, err := data.UUID(id)
	if err != nil {
		return err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteApplicationStatus(ctx, sqlc.DeleteApplicationStatusParams{
		ID:     sid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("store.DeleteApplicationStatus: %w", err)
	}
	return nil
}

func (s *Store) CountApplicationsUsingStatus(ctx context.Context, statusID, userID string) (int64, error) {
	sid, err := data.UUID(statusID)
	if err != nil {
		return 0, err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return 0, err
	}
	count, err := s.queries.CountApplicationsUsingStatus(ctx, sqlc.CountApplicationsUsingStatusParams{
		StatusID: sid,
		UserID:   uid,
	})
	if err != nil {
		return 0, fmt.Errorf("store.CountApplicationsUsingStatus: %w", err)
	}
	return count, nil
}
