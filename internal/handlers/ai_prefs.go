package handlers

import (
	"context"
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

// credLister is the subset of credstore.CredentialStore used by AIPrefsHandler.
// ponytail: minimal interface — only ListProviders needed here.
type credLister interface {
	ListProviders(ctx context.Context, userID string) ([]string, error)
}

type AIPrefsHandler struct {
	prefs providers.UserAIPrefsProvider
	creds credLister // nil-safe: returns empty slice when unset
}

func NewAIPrefsHandler(prefs providers.UserAIPrefsProvider, creds credLister) *AIPrefsHandler {
	return &AIPrefsHandler{prefs: prefs, creds: creds}
}

type aiPrefsResponse struct {
	SuitabilityModel    string   `json:"suitabilityModel"`
	AvailableModels     []string `json:"availableModels"`
	ConfiguredProviders []string `json:"configuredProviders"`
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

	var configured []string
	if h.creds != nil {
		configured, err = h.creds.ListProviders(r.Context(), session.UserID)
		if err != nil {
			slog.Error("list credential providers failed", slog.Any("err", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	if configured == nil {
		configured = []string{}
	}

	resp := aiPrefsResponse{
		SuitabilityModel:    model,
		AvailableModels:     availableModels,
		ConfiguredProviders: configured,
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
