package ingest_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

type canonicalSaver struct{}

func (canonicalSaver) SaveCanonical(_ context.Context, job dto.Job) (dto.Job, string, error) {
	if job.URL == "https://example.com/conflict" {
		return dto.Job{}, "", providers.ErrCanonicalConflict
	}
	job.ID = "job-1"
	return job, "new", nil
}

func TestIngestJobsKeepsValidJobsAfterRejection(t *testing.T) {
	ing := ingest.New(canonicalSaver{}, nil)
	results, err := ing.IngestJobs(t.Context(), []dto.Job{
		{Title: "Conflict", URL: "https://example.com/conflict"},
		{Title: "Invalid URL", URL: "file:///tmp/job"},
		{Title: "Valid", URL: "https://example.com/valid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[0].Status != "rejected" || results[1].Status != "rejected" || results[2].Status != "new" {
		t.Fatalf("results = %+v", results)
	}
}
