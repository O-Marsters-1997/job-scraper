package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

type ApplicationStatusHandler struct {
	statuses providers.ApplicationStatusProvider
}

func NewApplicationStatusHandler(statuses providers.ApplicationStatusProvider) *ApplicationStatusHandler {
	return &ApplicationStatusHandler{statuses: statuses}
}

func (h *ApplicationStatusHandler) ListApplicationStatuses(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	statuses, err := h.statuses.ListApplicationStatusesByUser(r.Context(), session.UserID)
	if err != nil {
		slog.Error("list application statuses failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statuses)
}

func (h *ApplicationStatusHandler) CreateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body struct {
		Name   string `json:"name"`
		Colour string `json:"colour"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Colour == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s, err := h.statuses.CreateApplicationStatus(r.Context(), session.UserID, body.Name, body.Colour)
	if err != nil {
		slog.Error("create application status failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(s)
}

func (h *ApplicationStatusHandler) UpdateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		Name   string `json:"name"`
		Colour string `json:"colour"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Colour == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s, err := h.statuses.UpdateApplicationStatus(r.Context(), id, session.UserID, body.Name, body.Colour)
	if err != nil {
		slog.Error("update application status failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s)
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
