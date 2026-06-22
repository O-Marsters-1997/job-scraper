package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func toUpsertParams(j dto.Job) pgsqlc.UpsertJobParams {
	return pgsqlc.UpsertJobParams{
		Title:           j.Title,
		Location:        j.Location,
		Url:             j.URL,
		CompanySlug:     j.CompanySlug,
		Source:          j.Source,
		UpdatedAt:       pgtype.Timestamptz{Time: j.UpdatedAt, Valid: true},
		Description:     j.Description,
		SalaryRaw:       j.SalaryRaw,
		WorkArrangement: j.WorkArrangement,
	}
}

func toUpsertBatchParams(jobs []dto.Job) []pgsqlc.UpsertJobsParams {
	params := make([]pgsqlc.UpsertJobsParams, len(jobs))
	for i, j := range jobs {
		params[i] = pgsqlc.UpsertJobsParams{
			Title:           j.Title,
			Location:        j.Location,
			Url:             j.URL,
			CompanySlug:     j.CompanySlug,
			Source:          j.Source,
			UpdatedAt:       pgtype.Timestamptz{Time: j.UpdatedAt, Valid: true},
			Description:     j.Description,
			SalaryRaw:       j.SalaryRaw,
			WorkArrangement: j.WorkArrangement,
		}
	}
	return params
}

func fromRow(row pgsqlc.Job) dto.Job {
	return dto.Job{
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		Description:     row.Description,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
}

func fromListRow(row pgsqlc.ListJobsRow) dto.Job {
	j := dto.Job{
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		Description:     row.Description,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
	if row.RelevanceScore.Valid {
		v := int(row.RelevanceScore.Int32)
		j.RelevanceScore = &v
	}
	if row.SuitabilityScore.Valid {
		v := int(row.SuitabilityScore.Int32)
		j.SuitabilityScore = &v
	}
	return j
}

func (db *DB) Save(ctx context.Context, jobs []dto.Job) error {
	if len(jobs) == 0 {
		return nil
	}
	if len(jobs) == 1 {
		_, err := db.queries.UpsertJob(ctx, toUpsertParams(jobs[0]))
		if err != nil {
			return fmt.Errorf("db.Save: %w", err)
		}
		slog.Debug("job saved",
			slog.String("url", jobs[0].URL),
		)
		return nil
	}
	results := db.queries.UpsertJobs(ctx, toUpsertBatchParams(jobs))
	defer func() { _ = results.Close() }()

	var errs []error
	results.Exec(func(i int, err error) {
		if err != nil {
			slog.Error("job save failed",
				slog.String("url", jobs[i].URL),
				slog.Any("err", err),
			)
			errs = append(errs, err)
		}
	})

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("db.Save: %w", err)
	}
	slog.Debug("jobs saved",
		slog.Int("count", len(jobs)),
	)
	return nil
}

func (db *DB) NewURLs(ctx context.Context, urls []string) ([]string, error) {
	existing, err := db.queries.ExistingURLs(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("db.NewURLs: %w", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, u := range existing {
		known[u] = struct{}{}
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if _, ok := known[u]; !ok {
			out = append(out, u)
		}
	}
	return out, nil
}

func (db *DB) List(ctx context.Context, userID string) ([]dto.Job, error) {
	var uid pgtype.UUID
	if userID != "" {
		var err error
		uid, err = parseUUID(userID)
		if err != nil {
			return nil, fmt.Errorf("db.List: %w", err)
		}
	}
	rows, err := db.queries.ListJobs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.List: %w", err)
	}
	jobs := make([]dto.Job, len(rows))
	for i, row := range rows {
		jobs[i] = fromListRow(row)
	}
	return jobs, nil
}

func (db *DB) ListSince(ctx context.Context, since time.Time) ([]dto.Job, error) {
	rows, err := db.queries.ListJobsSince(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("db.ListSince: %w", err)
	}
	jobs := make([]dto.Job, len(rows))
	for i, row := range rows {
		jobs[i] = fromRow(row)
	}
	return jobs, nil
}
