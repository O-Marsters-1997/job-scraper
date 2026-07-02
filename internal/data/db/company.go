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

func fromCompany(row pgsqlc.Company) dto.Company {
	return dto.Company{
		ID:          row.ID.String(),
		Slug:        row.Slug,
		Name:        row.Name,
		ATSSource:   row.AtsSource.String,
		ATSToken:    row.AtsToken.String,
		FirstSeenAt: row.FirstSeenAt.Time,
	}
}

func (db *DB) UpsertCompany(ctx context.Context, slug, name, atsSource, atsToken string) (dto.Company, error) {
	row, err := db.queries.UpsertCompany(ctx, pgsqlc.UpsertCompanyParams{
		Slug:      slug,
		Name:      name,
		AtsSource: pgtype.Text{String: atsSource, Valid: atsSource != ""},
		AtsToken:  pgtype.Text{String: atsToken, Valid: atsToken != ""},
	})
	if err != nil {
		return dto.Company{}, fmt.Errorf("db.UpsertCompany: %w", err)
	}
	return fromCompany(row), nil
}

func (db *DB) GetCompany(ctx context.Context, id string) (dto.Company, error) {
	cid, err := parseUUID(id)
	if err != nil {
		return dto.Company{}, err
	}
	row, err := db.queries.GetCompany(ctx, cid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.Company{}, providers.ErrNotFound
		}
		return dto.Company{}, fmt.Errorf("db.GetCompany: %w", err)
	}
	return fromCompany(row), nil
}

func (db *DB) ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListCompaniesForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListCompaniesForUser: %w", err)
	}
	out := make([]dto.Company, len(rows))
	for i, r := range rows {
		c := dto.Company{
			ID:          r.ID.String(),
			Slug:        r.Slug,
			Name:        r.Name,
			ATSSource:   r.AtsSource.String,
			ATSToken:    r.AtsToken.String,
			FirstSeenAt: r.FirstSeenAt.Time,
			JobCount:    int(r.JobCount),
			Tracked:     r.Tracked,
		}
		if r.TargetID.Valid {
			c.TargetID = r.TargetID.String()
			c.CheckIntervalMinutes = int(r.CheckIntervalMinutes.Int32)
		}
		if r.LastCheckedAt.Valid {
			t := r.LastCheckedAt.Time
			c.LastCheckedAt = &t
		}
		out[i] = c
	}
	return out, nil
}
