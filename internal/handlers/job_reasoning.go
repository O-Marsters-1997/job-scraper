package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/score"
)

// reasoningDB is the subset of *db.DB used by JobReasoningHandler.
type reasoningDB interface {
	GetJobScore(ctx context.Context, jobID, userID string) (dto.JobScore, error)
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)
	GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
	GetUserAIPrefs(ctx context.Context, userID string) (dto.UserAIPrefs, error)
	UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, sc int, reasoning string, matched, missing []string) error
}

type JobReasoningHandler struct {
	db    reasoningDB
	creds credstore.CredentialStore
}

func NewJobReasoningHandler(db reasoningDB, creds credstore.CredentialStore) *JobReasoningHandler {
	return &JobReasoningHandler{db: db, creds: creds}
}

// PostJobReasoning generates on-demand reasoning for an already-scored job.
// It re-scores the job using the user's configured reasoning model (default sonnet),
// overwriting the provisional ingest-time score with the authoritative result.
// If reasoning already exists, the cached result is returned without a new call.
func (h *JobReasoningHandler) PostJobReasoning(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	jobID := chi.URLParam(r, "id")

	existing, err := h.db.GetJobScore(r.Context(), jobID, session.UserID)
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			http.Error(w, "job not yet scored", http.StatusUnprocessableEntity)
			return
		}
		slog.Error("get job score failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existing.SuitabilitySkipped || existing.SuitabilityScore == nil {
		http.Error(w, "job not scored — skipped or pending", http.StatusUnprocessableEntity)
		return
	}

	// Return cached reasoning — no new Claude call.
	if existing.Reasoning != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(existing)
		return
	}

	job, err := h.db.GetJob(r.Context(), jobID, session.UserID)
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		slog.Error("get job failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	cfg, err := h.db.GetSearchConfig(r.Context(), session.UserID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		slog.Error("get search config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	reasoningModel := score.DefaultReasoningModel
	prefs, err := h.db.GetUserAIPrefs(r.Context(), session.UserID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		slog.Error("get ai prefs failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err == nil && prefs.ReasoningModel != "" {
		reasoningModel = prefs.ReasoningModel
	}

	provider := ingest.ProviderForModel(reasoningModel)
	apiKey, err := h.creds.Get(r.Context(), session.UserID, provider)
	if err != nil {
		if errors.Is(err, credstore.ErrNotFound) {
			http.Error(w, "scoring not enabled — configure an API key", http.StatusUnprocessableEntity)
			return
		}
		slog.Error("get credential failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	scorer := score.NewClaudeScorer(apiKey)
	result, err := scorer.Score(r.Context(), job, cfg, reasoningModel)
	if err != nil {
		slog.Error("reasoning score failed", slog.String("job_id", jobID), slog.Any("err", err))
		http.Error(w, "scoring failed", http.StatusBadGateway)
		return
	}

	if err := h.db.UpsertJobScoreSuitability(r.Context(), jobID, session.UserID, result.Score, result.Rationale, result.Matched, result.Missing); err != nil {
		slog.Error("upsert reasoning failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	updated, err := h.db.GetJobScore(r.Context(), jobID, session.UserID)
	if err != nil {
		slog.Error("re-fetch job score failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}
