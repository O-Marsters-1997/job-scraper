package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
)

func fromApplicationStatus(s pgsqlc.ApplicationStatus) dto.ApplicationStatus {
	return dto.ApplicationStatus{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		Name:      s.Name,
		Colour:    s.Colour,
		CreatedAt: s.CreatedAt.Time,
	}
}

func (db *DB) SeedDefaultStatuses(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.SeedDefaultStatuses(ctx, uid); err != nil {
		return fmt.Errorf("db.SeedDefaultStatuses: %w", err)
	}
	return nil
}

func (db *DB) CreateApplicationStatus(ctx context.Context, userID, name, colour string) (dto.ApplicationStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	s, err := db.queries.CreateApplicationStatus(ctx, pgsqlc.CreateApplicationStatusParams{
		UserID: uid,
		Name:   name,
		Colour: colour,
	})
	if err != nil {
		return dto.ApplicationStatus{}, fmt.Errorf("db.CreateApplicationStatus: %w", err)
	}
	return fromApplicationStatus(s), nil
}

func (db *DB) ListApplicationStatusesByUser(ctx context.Context, userID string) ([]dto.ApplicationStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListApplicationStatusesByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListApplicationStatusesByUser: %w", err)
	}
	out := make([]dto.ApplicationStatus, len(rows))
	for i, row := range rows {
		out[i] = fromApplicationStatus(row)
	}
	return out, nil
}

func (db *DB) UpdateApplicationStatus(ctx context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	s, err := db.queries.UpdateApplicationStatus(ctx, pgsqlc.UpdateApplicationStatusParams{
		ID:     sid,
		UserID: uid,
		Name:   name,
		Colour: colour,
	})
	if err != nil {
		return dto.ApplicationStatus{}, fmt.Errorf("db.UpdateApplicationStatus: %w", err)
	}
	return fromApplicationStatus(s), nil
}

func (db *DB) DeleteApplicationStatus(ctx context.Context, id, userID string) error {
	sid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.DeleteApplicationStatus(ctx, pgsqlc.DeleteApplicationStatusParams{
		ID:     sid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("db.DeleteApplicationStatus: %w", err)
	}
	return nil
}

func (db *DB) CountApplicationsUsingStatus(ctx context.Context, statusID, userID string) (int64, error) {
	sid, err := parseUUID(statusID)
	if err != nil {
		return 0, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	count, err := db.queries.CountApplicationsUsingStatus(ctx, pgsqlc.CountApplicationsUsingStatusParams{
		StatusID: sid,
		UserID:   uid,
	})
	if err != nil {
		return 0, fmt.Errorf("db.CountApplicationsUsingStatus: %w", err)
	}
	return count, nil
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
