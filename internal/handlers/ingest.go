package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

type IngestHandler struct {
	ing *ingest.Ingester
}

func NewIngestHandler(ing *ingest.Ingester) *IngestHandler {
	return &IngestHandler{ing: ing}
}

func (h *IngestHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	job, ok := DecodeJSON[dto.Job](w, r)
	if !ok {
		return
	}
	results, err := h.ing.IngestJobs(r.Context(), []dto.Job{job})
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, results[0])
}

func (h *IngestHandler) IngestBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Jobs []dto.Job `json:"jobs"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&request); err != nil || len(request.Jobs) == 0 || len(request.Jobs) > 100 {
		WriteError(w, r, apperr.Invalid("invalid batch"))
		return
	}
	results, err := h.ing.IngestJobs(r.Context(), request.Jobs)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, struct {
		Results []ingest.Result `json:"results"`
	}{Results: results})
}
