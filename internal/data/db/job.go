package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	repository "github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

func (db *DB) UpsertJob(ctx context.Context, job sources.Job) error {
	_, err := repository.New(db.pool).UpsertJob(ctx, repository.UpsertJobParams{
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
	params := make([]repository.UpsertJobsParams, len(jobs))
	for i, job := range jobs {
		params[i] = repository.UpsertJobsParams{
			Title:       job.Title,
			Location:    job.Location,
			Url:         job.URL,
			CompanySlug: job.CompanySlug,
			Source:      job.Source,
			UpdatedAt:   pgtype.Timestamptz{Time: job.UpdatedAt, Valid: true},
		}
	}
	results := repository.New(db.pool).UpsertJobs(ctx, params)
	defer results.Close()
	var errs []error
	results.Exec(func(_ int, err error) {
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}
