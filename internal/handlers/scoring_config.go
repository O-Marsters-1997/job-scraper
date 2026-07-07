package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type ScoringConfigHandler struct {
	configs providers.SearchConfigProvider
}

func NewScoringConfigHandler(configs providers.SearchConfigProvider) *ScoringConfigHandler {
	return &ScoringConfigHandler{configs: configs}
}

type scoringConfigResponse struct {
	SuitabilityRubric     string   `json:"suitabilityRubric"`
	NotifyThreshold       int      `json:"notifyThreshold"`
	ExcludedTitleKeywords []string `json:"excludedTitleKeywords"`
	ExcludedCompanies     []string `json:"excludedCompanies"`
	ExcludedSeniority     []string `json:"excludedSeniority"`
	ExcludedLocations     []string `json:"excludedLocations"`
}

func toScoringConfigResponse(cfg dto.SearchConfig) scoringConfigResponse {
	return scoringConfigResponse{
		SuitabilityRubric:     cfg.SuitabilityRubric,
		NotifyThreshold:       cfg.NotifyThreshold,
		ExcludedTitleKeywords: nonNilStrings(cfg.ExcludedTitleKeywords),
		ExcludedCompanies:     nonNilStrings(cfg.ExcludedCompanies),
		ExcludedSeniority:     nonNilStrings(cfg.ExcludedSeniority),
		ExcludedLocations:     nonNilStrings(cfg.ExcludedLocations),
	}
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (h *ScoringConfigHandler) GetScoringConfig(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	cfg, err := h.configs.GetSearchConfig(r.Context(), session.UserID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		slog.Error("get scoring config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toScoringConfigResponse(cfg))
}

func (h *ScoringConfigHandler) UpdateScoringConfig(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body scoringConfigResponse
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	seniority := cleanList(body.ExcludedSeniority)
	for _, level := range seniority {
		if !slices.Contains(score.SeniorityLevels, level) {
			http.Error(w, "unknown seniority level: "+level, http.StatusBadRequest)
			return
		}
	}

	cfg := dto.SearchConfig{
		UserID:                session.UserID,
		SuitabilityRubric:     body.SuitabilityRubric,
		NotifyThreshold:       body.NotifyThreshold,
		ExcludedTitleKeywords: cleanList(body.ExcludedTitleKeywords),
		ExcludedCompanies:     cleanList(body.ExcludedCompanies),
		ExcludedSeniority:     seniority,
		ExcludedLocations:     cleanList(body.ExcludedLocations),
	}
	updated, err := h.configs.UpsertSearchConfig(r.Context(), cfg)
	if err != nil {
		slog.Error("upsert scoring config failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toScoringConfigResponse(updated))
}

// cleanList trims and lowercases each entry, dropping any that are empty.
func cleanList(items []string) []string {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}
