package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

type Handler struct {
	jobs               providers.JobProvider
	users              providers.UserProvider
	sessions           providers.SessionProvider
	applicationStatuses providers.ApplicationStatusProvider
	applications       providers.ApplicationProvider
}

func New(
	jobs providers.JobProvider,
	users providers.UserProvider,
	sessions providers.SessionProvider,
	applicationStatuses providers.ApplicationStatusProvider,
	applications providers.ApplicationProvider,
) *Handler {
	return &Handler{
		jobs:                jobs,
		users:               users,
		sessions:            sessions,
		applicationStatuses: applicationStatuses,
		applications:        applications,
	}
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

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobs.List(r.Context())
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
