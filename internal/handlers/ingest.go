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
	if err := h.ing.Ingest(r.Context(), []dto.Job{job}); err != nil {
		http.Error(w, `{"error":"ingest failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ingested":1}`))
}
