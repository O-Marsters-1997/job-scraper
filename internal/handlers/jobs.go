package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

type JobHandler struct {
	jobs providers.JobProvider
}

func NewJobHandler(jobs providers.JobProvider) *JobHandler {
	return &JobHandler{jobs: jobs}
}

func newSessionCookie(id string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    id,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
		Path:     "/",
		MaxAge:   maxAge,
	}
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobs.List(r.Context())
	if err != nil {
		slog.Error("list jobs failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		slog.Error("encode jobs failed",
			slog.Any("err", err),
		)
	}
}
