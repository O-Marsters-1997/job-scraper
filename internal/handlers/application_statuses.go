package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

// ApplicationStatusHandler holds DeleteApplicationStatus, which stays a
// misfit per ADR 0020: it returns 409 plus an in-use count, a shape the
// generic adapters don't express.
type ApplicationStatusHandler struct {
	statuses providers.ApplicationStatusProvider
}

func NewApplicationStatusHandler(statuses providers.ApplicationStatusProvider) *ApplicationStatusHandler {
	return &ApplicationStatusHandler{statuses: statuses}
}

func (h *ApplicationStatusHandler) DeleteApplicationStatus(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")

	count, err := h.statuses.CountApplicationsUsingStatus(r.Context(), id, session.UserID)
	if err != nil {
		slog.Error("count applications using status failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "status is in use",
			"count": count,
		})
		return
	}

	if err := h.statuses.DeleteApplicationStatus(r.Context(), id, session.UserID); err != nil {
		slog.Error("delete application status failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
