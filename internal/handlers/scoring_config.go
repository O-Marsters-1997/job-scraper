package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type ScoringConfigHandler struct {
	configs providers.SearchConfigProvider
}

func NewScoringConfigHandler(configs providers.SearchConfigProvider) *ScoringConfigHandler {
	return &ScoringConfigHandler{configs: configs}
}

type scoringConfigResponse struct {
	SuitabilityRubric string `json:"suitabilityRubric"`
	RelevanceCutoff   int    `json:"relevanceCutoff"`
	NotifyThreshold   int    `json:"notifyThreshold"`
}

func (h *ScoringConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	cfg, err := h.configs.GetSearchConfig(r.Context(), session.UserID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		slog.Error("get scoring config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	resp := scoringConfigResponse{
		SuitabilityRubric: cfg.SuitabilityRubric,
		RelevanceCutoff:   cfg.RelevanceCutoff,
		NotifyThreshold:   cfg.NotifyThreshold,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ScoringConfigHandler) Put(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body struct {
		SuitabilityRubric string `json:"suitabilityRubric"`
		RelevanceCutoff   int    `json:"relevanceCutoff"`
		NotifyThreshold   int    `json:"notifyThreshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	cfg := dto.SearchConfig{
		UserID:            session.UserID,
		SuitabilityRubric: body.SuitabilityRubric,
		RelevanceCutoff:   body.RelevanceCutoff,
		NotifyThreshold:   body.NotifyThreshold,
	}
	if _, err := h.configs.UpsertSearchConfig(r.Context(), cfg); err != nil {
		slog.Error("upsert scoring config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
