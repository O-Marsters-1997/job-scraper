package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

var availableModels = []string{
	"claude-haiku-4-5-20251001",
	"claude-sonnet-4-6",
	"claude-opus-4-8",
}

const defaultSuitabilityModel = "claude-haiku-4-5-20251001"

type AIPrefsHandler struct {
	prefs providers.UserAIPrefsProvider
}

func NewAIPrefsHandler(prefs providers.UserAIPrefsProvider) *AIPrefsHandler {
	return &AIPrefsHandler{prefs: prefs}
}

type aiPrefsResponse struct {
	SuitabilityModel string   `json:"suitabilityModel"`
	AvailableModels  []string `json:"availableModels"`
}

func (h *AIPrefsHandler) Get(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	prefs, err := h.prefs.GetUserAIPrefs(r.Context(), session.UserID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		slog.Error("get ai prefs failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	model := defaultSuitabilityModel
	if err == nil {
		model = prefs.SuitabilityModel
	}
	resp := aiPrefsResponse{
		SuitabilityModel: model,
		AvailableModels:  availableModels,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AIPrefsHandler) Put(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body struct {
		SuitabilityModel string `json:"suitabilityModel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !isValidModel(body.SuitabilityModel) {
		http.Error(w, "invalid model", http.StatusBadRequest)
		return
	}
	if _, err := h.prefs.UpsertUserAIPrefs(r.Context(), session.UserID, body.SuitabilityModel); err != nil {
		slog.Error("upsert ai prefs failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func isValidModel(model string) bool {
	for _, m := range availableModels {
		if m == model {
			return true
		}
	}
	return false
}
