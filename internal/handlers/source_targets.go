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
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type SourceTargetHandler struct {
	targets providers.SourceTargetProvider
	q       queue.JobQueue
}

func NewSourceTargetHandler(targets providers.SourceTargetProvider, q queue.JobQueue) *SourceTargetHandler {
	return &SourceTargetHandler{targets: targets, q: q}
}

func (h *SourceTargetHandler) Sources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sources.Sources())
}

// ResolveBoard turns a direct ATS board URL into a {source, value} pair the client
// can use to pre-fill the add-target form. Returns 422 if no ATS board is recognised.
func (h *SourceTargetHandler) ResolveBoard(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	source, token, ok := detect.ResolveBoard(rawURL)
	if !ok {
		http.Error(w, "could not resolve an ATS board from that URL", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Source string `json:"source"`
		Value  string `json:"value"`
	}{Source: source, Value: token})
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
		Source    string            `json:"source"`
		Value     string            `json:"value"`
		Enabled   *bool             `json:"enabled"`
		Filters   map[string]string `json:"filters"`
		ScrapeNow bool              `json:"scrape_now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Source == "" || body.Value == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Cross-role guard: an ATS board URL pasted into a discovery source belongs under
	// Tracked companies, not here. (Discovery values are keywords or aggregator URLs.)
	if role, _ := sources.SourceRole(body.Source); role == sources.RoleDiscovery {
		if t := detect.Detect(body.Value); t != detect.UnknownHTML && t != detect.Aggregator {
			http.Error(w, "that looks like an ATS board — add it under Tracked companies", http.StatusBadRequest)
			return
		}
	}

	// Validate against the registry. Filter sources get their own path; board/url
	// sources keep the existing URL-prefix check.
	if fields, isFilter := sources.LookupFilterFields(body.Source); isFilter {
		for k := range body.Filters {
			if !isKnownFilterField(k, fields) {
				http.Error(w, "unknown filter key: "+k, http.StatusBadRequest)
				return
			}
		}
		for _, f := range fields {
			if f.Required && body.Filters[f.Name] == "" {
				http.Error(w, "missing required filter: "+f.Name, http.StatusBadRequest)
				return
			}
		}
	} else {
		urlPrefix, isURL, ok := sources.LookupSource(body.Source)
		if !ok {
			http.Error(w, "unsupported source", http.StatusBadRequest)
			return
		}
		if isURL && !strings.HasPrefix(body.Value, urlPrefix) {
			http.Error(w, "value must be a URL starting with "+urlPrefix, http.StatusBadRequest)
			return
		}
		// ATS board slot: value must be a bare token, not a URL.
		if !isURL && (strings.Contains(body.Value, "://") || strings.Contains(body.Value, "/")) {
			http.Error(w, "enter just the board token, e.g. acmecorp, not the full URL", http.StatusBadRequest)
			return
		}
	}

	if body.Filters == nil {
		body.Filters = map[string]string{}
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	t, err := h.targets.CreateSourceTarget(r.Context(), session.UserID, body.Source, body.Value, enabled, body.Filters)
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

	if body.ScrapeNow && enabled && h.q != nil {
		if err := h.q.EnqueueScrapeRequest(r.Context(), dto.ScrapeRequest{Target: t}); err != nil {
			slog.Error("enqueue scrape request failed", slog.Any("err", err))
		}
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

func isKnownFilterField(key string, fields []sources.FilterField) bool {
	for _, f := range fields {
		if f.Name == key {
			return true
		}
	}
	return false
}
