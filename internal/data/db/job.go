package db

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func toUpsertParams(j dto.Job) pgsqlc.UpsertJobParams {
	return pgsqlc.UpsertJobParams{
		Title:       j.Title,
		Location:    j.Location,
		Url:         j.URL,
		CompanySlug: j.CompanySlug,
		Source:      j.Source,
		UpdatedAt:   pgtype.Timestamptz{Time: j.UpdatedAt, Valid: true},
	}
}

func toUpsertBatchParams(jobs []dto.Job) []pgsqlc.UpsertJobsParams {
	params := make([]pgsqlc.UpsertJobsParams, len(jobs))
	for i, j := range jobs {
		params[i] = pgsqlc.UpsertJobsParams{
			Title:       j.Title,
			Location:    j.Location,
			Url:         j.URL,
			CompanySlug: j.CompanySlug,
			Source:      j.Source,
			UpdatedAt:   pgtype.Timestamptz{Time: j.UpdatedAt, Valid: true},
		}
	}
	return params
}

func fromRow(row pgsqlc.Job) dto.Job {
	return dto.Job{
		ID:          row.ID.String(),
		Title:       row.Title,
		Location:    row.Location,
		URL:         row.Url,
		CompanySlug: row.CompanySlug,
		Source:      row.Source,
		UpdatedAt:   row.UpdatedAt.Time,
		ScrapedAt:   row.ScrapedAt.Time,
	}
}

func (db *DB) Save(ctx context.Context, jobs []dto.Job) error {
	if len(jobs) == 0 {
		slog.Info("no jobs to save")
		return nil
	}
	if len(jobs) == 1 {
		_, err := db.queries.UpsertJob(ctx, toUpsertParams(jobs[0]))
		return err
	}
	results := db.queries.UpsertJobs(ctx, toUpsertBatchParams(jobs))
	defer func() { _ = results.Close() }()

	var errs []error
	results.Exec(func(i int, err error) {
		if err != nil {
			slog.Error("job save failed", slog.String("url", jobs[i].URL), slog.Any("err", err))
			errs = append(errs, err)
		}
	})

	return errors.Join(errs...)
}

func (db *DB) NewURLs(ctx context.Context, urls []string) ([]string, error) {
	existing, err := db.queries.ExistingURLs(ctx, urls)
	if err != nil {
		return nil, err
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

func (db *DB) List(ctx context.Context) ([]dto.Job, error) {
	rows, err := db.queries.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	jobs := make([]dto.Job, len(rows))
	for i, row := range rows {
		jobs[i] = fromRow(row)
	}
	return jobs, nil
}
