package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

// Ingest and IngestBatch authenticate by service token, not a session, so
// their decode step never calls userID.
func Ingest(ing *ingest.Ingester) http.HandlerFunc {
	return Handle(
		decodeBody[dto.Job],
		func(ctx context.Context, job dto.Job) (ingest.Result, error) {
			results, err := ing.IngestJobs(ctx, []dto.Job{job})
			if err != nil {
				return ingest.Result{}, err
			}
			return results[0], nil
		},
		respondJSON[ingest.Result](http.StatusOK),
	)
}

type ingestBatchInput struct {
	Jobs []dto.Job `json:"jobs"`
}

type ingestBatchView struct {
	Results []ingest.Result `json:"results"`
}

func IngestBatch(ing *ingest.Ingester) http.HandlerFunc {
	return Handle(
		func(r *http.Request) (ingestBatchInput, error) {
			var in ingestBatchInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Jobs) == 0 || len(in.Jobs) > 100 {
				return ingestBatchInput{}, apperr.Invalid("invalid batch")
			}
			return in, nil
		},
		func(ctx context.Context, in ingestBatchInput) (ingestBatchView, error) {
			results, err := ing.IngestJobs(ctx, in.Jobs)
			if err != nil {
				return ingestBatchView{}, err
			}
			return ingestBatchView{Results: results}, nil
		},
		respondJSON[ingestBatchView](http.StatusOK),
	)
}
