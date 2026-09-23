package handlers

import (
	"encoding/json"
	"net/http"

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
	var job dto.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	results, err := h.ing.IngestJobs(r.Context(), []dto.Job{job})
	if err != nil {
		http.Error(w, `{"error":"ingest failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results[0])
}

func (h *IngestHandler) IngestBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Jobs []dto.Job `json:"jobs"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&request); err != nil || len(request.Jobs) == 0 || len(request.Jobs) > 100 {
		http.Error(w, `{"error":"invalid batch"}`, http.StatusBadRequest)
		return
	}
	results, err := h.ing.IngestJobs(r.Context(), request.Jobs)
	if err != nil {
		http.Error(w, `{"error":"ingest failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Results []ingest.Result `json:"results"`
	}{Results: results})
}
