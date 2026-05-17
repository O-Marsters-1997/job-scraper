package db

import (
	"context"
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
	return db.queries.SeedDefaultStatuses(ctx, uid)
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
		return dto.ApplicationStatus{}, err
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
		return nil, err
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
		return dto.ApplicationStatus{}, err
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
	return db.queries.DeleteApplicationStatus(ctx, pgsqlc.DeleteApplicationStatusParams{
		ID:     sid,
		UserID: uid,
	})
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
	return db.queries.CountApplicationsUsingStatus(ctx, pgsqlc.CountApplicationsUsingStatusParams{
		StatusID: sid,
		UserID:   uid,
	})
}

func toOptionalDate(s fp.Option[string]) pgtype.Date {
	if s.IsNone() {
		return pgtype.Date{}
	}
	var d pgtype.Date
	_ = d.Scan(s.Unwrap())
	return d
}

// fromOptionalDate converts a pgtype.Date to *time.Time.
func fromOptionalDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}
