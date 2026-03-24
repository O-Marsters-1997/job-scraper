package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

type Handler struct {
	db providers.JobProvider
}

func New(db providers.JobProvider) *Handler {
	return &Handler{db: db}
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.List(r.Context())
	if err != nil {
		slog.Error("list jobs failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		slog.Error("encode jobs failed", slog.Any("err", err))
	}
}
