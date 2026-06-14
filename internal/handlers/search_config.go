package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SearchConfigHandler struct {
	db providers.SearchConfigProvider
}

func NewSearchConfigHandler(db providers.SearchConfigProvider) *SearchConfigHandler {
	return &SearchConfigHandler{db: db}
}

func (h *SearchConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	cfg, err := h.db.GetSearchConfig(r.Context(), session.UserID)
	if err != nil {
		slog.Error("get search config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

func (h *SearchConfigHandler) Put(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	var cfg dto.SearchConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	cfg.UserID = session.UserID

	updated, err := h.db.UpsertSearchConfig(r.Context(), cfg)
	if err != nil {
		slog.Error("upsert search config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}
