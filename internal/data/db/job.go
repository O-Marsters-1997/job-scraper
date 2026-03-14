package db

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/sources"
)

func (db *DB) UpsertJobs(ctx context.Context, jobs []sources.Job) error {
	// TODO: Implement
	return nil
}

func (db *DB) UpsertJob(ctx context.Context, job sources.Job) error {
	// TODO: Implement
	return nil
}
