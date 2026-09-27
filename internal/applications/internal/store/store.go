// Package store is the applications context's Postgres store: applications
// and application_statuses, plus a read-join onto jobs for the list view.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/applications/internal/store/sqlc"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
)

var (
	ErrApplicationExists = apperr.Conflict("application already exists for this job")
	ErrNotFound          = apperr.NotFound("not found")
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func toOptionalDate(s fp.Option[string]) (pgtype.Date, error) {
	if s.IsNone() {
		return pgtype.Date{}, nil
	}
	var d pgtype.Date
	if err := d.Scan(s.Unwrap()); err != nil {
		return pgtype.Date{}, fmt.Errorf("invalid applied date: %w", err)
	}
	return d, nil
}

func fromOptionalDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func fromApplication(a sqlc.Application) dto.Application {
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

func fromApplicationListRow(r sqlc.ListApplicationsByUserRow) dto.ApplicationWithDetails {
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

func fromApplicationListStatusRow(r sqlc.ListApplicationsByUserAndStatusRow) dto.ApplicationWithDetails {
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

func fromApplicationStatus(s sqlc.ApplicationStatus) dto.ApplicationStatus {
	return dto.ApplicationStatus{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		Name:      s.Name,
		Colour:    s.Colour,
		CreatedAt: s.CreatedAt.Time,
	}
}

func (s *Store) CreateApplication(ctx context.Context, userID string, input dto.CreateApplicationInput) (dto.Application, error) {
	uid, err := parseUUID(userID)
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
	a, err := s.queries.CreateApplication(ctx, sqlc.CreateApplicationParams{
		UserID:     uid,
		JobID:      jid,
		StatusID:   sid,
		Notes:      pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		AppliedAt:  appliedAt,
		SalaryInfo: pgtype.Text{String: input.SalaryInfo, Valid: input.SalaryInfo != ""},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.Application{}, ErrApplicationExists
		}
		return dto.Application{}, fmt.Errorf("store.CreateApplication: %w", err)
	}
	return fromApplication(a), nil
}

func (s *Store) ListApplicationsByUser(ctx context.Context, userID string) ([]dto.ApplicationWithDetails, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListApplicationsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListApplicationsByUser: %w", err)
	}
	out := make([]dto.ApplicationWithDetails, len(rows))
	for i, row := range rows {
		out[i] = fromApplicationListRow(row)
	}
	return out, nil
}

func (s *Store) ListApplicationsByUserAndStatus(ctx context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	sid, err := parseUUID(statusID)
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
		out[i] = fromApplicationListStatusRow(row)
	}
	return out, nil
}

func (s *Store) UpdateApplication(ctx context.Context, userID, id string, input dto.UpdateApplicationInput) (dto.Application, error) {
	aid, err := parseUUID(id)
	if err != nil {
		return dto.Application{}, err
	}
	uid, err := parseUUID(userID)
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
	a, err := s.queries.UpdateApplication(ctx, sqlc.UpdateApplicationParams{
		ID:         aid,
		UserID:     uid,
		StatusID:   sid,
		Notes:      pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		AppliedAt:  appliedAt,
		SalaryInfo: pgtype.Text{String: input.SalaryInfo, Valid: input.SalaryInfo != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.Application{}, ErrNotFound
		}
		return dto.Application{}, fmt.Errorf("store.UpdateApplication: %w", err)
	}
	return fromApplication(a), nil
}

func (s *Store) DeleteApplication(ctx context.Context, userID, id string) error {
	aid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
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
	rows, err := s.queries.GetApplicationsForJobs(ctx, sqlc.GetApplicationsForJobsParams{
		UserID:  uid,
		Column2: pgIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("store.GetApplicationsForJobs: %w", err)
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

func (s *Store) SeedDefaultStatuses(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.SeedDefaultStatuses(ctx, uid); err != nil {
		return fmt.Errorf("store.SeedDefaultStatuses: %w", err)
	}
	return nil
}

func (s *Store) CreateApplicationStatus(ctx context.Context, userID, name, colour string) (dto.ApplicationStatus, error) {
	uid, err := parseUUID(userID)
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
	return fromApplicationStatus(st), nil
}

func (s *Store) ListApplicationStatusesByUser(ctx context.Context, userID string) ([]dto.ApplicationStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListApplicationStatusesByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListApplicationStatusesByUser: %w", err)
	}
	out := make([]dto.ApplicationStatus, len(rows))
	for i, row := range rows {
		out[i] = fromApplicationStatus(row)
	}
	return out, nil
}

func (s *Store) UpdateApplicationStatus(ctx context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	uid, err := parseUUID(userID)
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
	return fromApplicationStatus(st), nil
}

func (s *Store) DeleteApplicationStatus(ctx context.Context, id, userID string) error {
	sid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
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
	sid, err := parseUUID(statusID)
	if err != nil {
		return 0, err
	}
	uid, err := parseUUID(userID)
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
