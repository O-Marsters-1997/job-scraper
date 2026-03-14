package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

func (db *DB) UpsertJob(ctx context.Context, job sources.Job) error {
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

func (db *DB) UpsertJobs(ctx context.Context, jobs []sources.Job) error {
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
	defer results.Close()
	var errs []error
	results.Exec(func(_ int, err error) {
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}
