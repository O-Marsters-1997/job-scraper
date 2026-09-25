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
	c := dto.Company{
		ID:                row.ID.String(),
		Slug:              row.Slug,
		Name:              row.Name,
		ATSSource:         row.AtsSource.String,
		ATSToken:          row.AtsToken.String,
		Domain:            row.Domain.String,
		LinkedInCompanyID: row.LinkedinCompanyID.String,
		FirstSeenAt:       row.FirstSeenAt.Time,
	}
	if row.LastCrawledAt.Valid {
		t := row.LastCrawledAt.Time
		c.LastCrawledAt = &t
	}
	return c
}

func (db *DB) UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	row, err := db.queries.UpsertCompany(ctx, pgsqlc.UpsertCompanyParams{
		Slug:              c.Slug,
		Name:              c.Name,
		AtsSource:         pgtype.Text{String: c.ATSSource, Valid: c.ATSSource != ""},
		AtsToken:          pgtype.Text{String: c.ATSToken, Valid: c.ATSToken != ""},
		Domain:            pgtype.Text{String: c.Domain, Valid: c.Domain != ""},
		LinkedinCompanyID: pgtype.Text{String: c.LinkedInCompanyID, Valid: c.LinkedInCompanyID != ""},
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
			ID:                   r.ID.String(),
			Slug:                 r.Slug,
			Name:                 r.Name,
			ATSSource:            r.AtsSource.String,
			ATSToken:             r.AtsToken.String,
			Domain:               r.Domain.String,
			LinkedInCompanyID:    r.LinkedinCompanyID.String,
			FirstSeenAt:          r.FirstSeenAt.Time,
			JobCount:             int(r.JobCount),
			Tracked:              r.Tracked,
			CheckIntervalMinutes: int(r.CheckIntervalMinutes.Int32),
		}
		if r.LastCheckedAt.Valid {
			t := r.LastCheckedAt.Time
			c.LastCheckedAt = &t
		}
		if r.LastCrawledAt.Valid {
			t := r.LastCrawledAt.Time
			c.LastCrawledAt = &t
		}
		out[i] = c
	}
	return out, nil
}

func (db *DB) SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, interval int) (dto.CompanyTracking, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	cid, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("begin company tracking: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.queries.WithTx(tx)
	row, err := queries.SetCompanyTracking(ctx, pgsqlc.SetCompanyTrackingParams{
		UserID: uid, CompanyID: cid, Enabled: enabled, CheckIntervalMinutes: int32(interval),
	})
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("db.SetCompanyTracking: %w", err)
	}
	if enabled {
		if err := queries.BackfillCompanyJobFingerprints(ctx, cid); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("backfill tracked company jobs: %w", err)
		}
		if err := queries.QueueTrackingScores(ctx, pgsqlc.QueueTrackingScoresParams{UserID: uid, CompanyID: cid}); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("queue tracked company scores: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("commit company tracking: %w", err)
	}
	return dto.CompanyTracking{
		UserID: row.UserID.String(), CompanyID: row.CompanyID.String(),
		Enabled: row.Enabled, CheckIntervalMinutes: int(row.CheckIntervalMinutes),
	}, nil
}

// ListCompaniesToCrawl returns up to limit companies with a known domain but
// no resolved ATS board, least-recently-crawled first, for the careers-page
// crawler to work through.
func (db *DB) ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error) {
	rows, err := db.queries.ListCompaniesToCrawl(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("db.ListCompaniesToCrawl: %w", err)
	}
	out := make([]dto.Company, len(rows))
	for i, r := range rows {
		out[i] = fromCompany(r)
	}
	return out, nil
}

func (db *DB) TouchCompanyCrawled(ctx context.Context, id string) error {
	cid, err := parseUUID(id)
	if err != nil {
		return err
	}
	if err := db.queries.TouchCompanyCrawled(ctx, cid); err != nil {
		return fmt.Errorf("db.TouchCompanyCrawled: %w", err)
	}
	return nil
}
