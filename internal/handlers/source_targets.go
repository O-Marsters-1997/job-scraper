package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type SourceTargetHandler struct {
	targets providers.SourceTargetProvider
}

func NewSourceTargetHandler(targets providers.SourceTargetProvider) *SourceTargetHandler {
	return &SourceTargetHandler{targets: targets}
}

func (h *SourceTargetHandler) List(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	ts, err := h.targets.ListSourceTargetsByUser(r.Context(), session.UserID)
	if err != nil {
		slog.Error("list source targets failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ts)
}

func (h *SourceTargetHandler) Create(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body struct {
		Source  string `json:"source"`
		Value   string `json:"value"`
		Enabled *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Source == "" || body.Value == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	urlPrefix, isURL, ok := sources.LookupSource(body.Source)
	if !ok {
		http.Error(w, "unsupported source", http.StatusBadRequest)
		return
	}
	if isURL && !strings.HasPrefix(body.Value, urlPrefix) {
		http.Error(w, "value must be a URL starting with "+urlPrefix, http.StatusBadRequest)
		return
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	t, err := h.targets.CreateSourceTarget(r.Context(), session.UserID, body.Source, body.Value, enabled)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"source target already exists"}`))
			return
		}
		slog.Error("create source target failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *SourceTargetHandler) Update(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t, err := h.targets.UpdateSourceTarget(r.Context(), id, session.UserID, body.Enabled)
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update source target failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

func (h *SourceTargetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.targets.DeleteSourceTarget(r.Context(), id, session.UserID); err != nil {
		slog.Error("delete source target failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
