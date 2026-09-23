package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/db"
)

type ScoresHandler struct {
	effects interface {
		GetScoringStatus(context.Context, string) (db.ScoringStatus, error)
		QueueRescore(context.Context, string) (int64, error)
	}
}

func NewScoresHandler(effects interface {
	GetScoringStatus(context.Context, string) (db.ScoringStatus, error)
	QueueRescore(context.Context, string) (int64, error)
}) *ScoresHandler {
	return &ScoresHandler{effects: effects}
}

func (h *ScoresHandler) Status(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	status, err := h.effects.GetScoringStatus(r.Context(), session.UserID)
	if err != nil {
		slog.Error("get scoring status failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (h *ScoresHandler) Rescore(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	queued, err := h.effects.QueueRescore(r.Context(), session.UserID)
	if err != nil {
		slog.Error("queue rescore failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Queued int64 `json:"queued"`
	}{Queued: queued})
}
