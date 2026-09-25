package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

func Ingest(ing *ingest.Ingester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, ok := decodeBody[dto.Job](w, r)
		if !ok {
			return
		}
		results, err := ing.IngestJobs(r.Context(), []dto.Job{job})
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, results[0])
	}
}

func IngestBatch(ing *ingest.Ingester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Jobs []dto.Job `json:"jobs"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&request); err != nil || len(request.Jobs) == 0 || len(request.Jobs) > 100 {
			writeError(w, r, apperr.Invalid("invalid batch"))
			return
		}
		results, err := ing.IngestJobs(r.Context(), request.Jobs)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Results []ingest.Result `json:"results"`
		}{Results: results})
	}
}
