package db

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (db *DB) UpsertJob(ctx context.Context, job dto.Job) error {
	slog.Debug("upserting job", slog.String("url", job.URL), slog.String("source", job.Source))
	_, err := pgsqlc.New(db.pool).UpsertJob(ctx, pgsqlc.UpsertJobParams{
		Title:       job.Title,
		Location:    job.Location,
		Url:         job.URL,
		CompanySlug: job.CompanySlug,
		Source:      job.Source,
		UpdatedAt:   pgtype.Timestamptz{Time: job.UpdatedAt, Valid: true},
	})
	return err
}

func (db *DB) FilterNewURLs(ctx context.Context, urls []string) ([]string, error) {
	existing, err := pgsqlc.New(db.pool).ExistingURLs(ctx, urls)
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

func (db *DB) UpsertJobs(ctx context.Context, jobs []dto.Job) error {
	slog.Debug("upserting jobs", slog.Int("count", len(jobs)))

	params := make([]pgsqlc.UpsertJobsParams, len(jobs))
	for i, job := range jobs {
		params[i] = pgsqlc.UpsertJobsParams{
			Title:       job.Title,
			Location:    job.Location,
			Url:         job.URL,
			CompanySlug: job.CompanySlug,
			Source:      job.Source,
			UpdatedAt:   pgtype.Timestamptz{Time: job.UpdatedAt, Valid: true},
		}
	}

	results := pgsqlc.New(db.pool).UpsertJobs(ctx, params)
	defer func() { _ = results.Close() }()

	var errs []error
	results.Exec(func(i int, err error) {
		if err != nil {
			slog.Error("job upsert failed", slog.String("url", jobs[i].URL), slog.Any("err", err))
			errs = append(errs, err)
		}
	})

	slog.Info("jobs upserted", slog.Int("count", len(jobs)-len(errs)))
	return errors.Join(errs...)
}
