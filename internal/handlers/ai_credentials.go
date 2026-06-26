package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/credstore"
)

type AICredentialsHandler struct {
	creds credstore.CredentialStore
}

func NewAICredentialsHandler(creds credstore.CredentialStore) *AICredentialsHandler {
	return &AICredentialsHandler{creds: creds}
}

func (h *AICredentialsHandler) UpsertCredential(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())

	var body struct {
		Provider string  `json:"provider"`
		APIKey   *string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Provider == "" {
		http.Error(w, "provider required", http.StatusBadRequest)
		return
	}

	if body.APIKey == nil {
		if err := h.creds.Delete(r.Context(), session.UserID, body.Provider); err != nil {
			slog.Error("delete credential failed", slog.Any("err", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		key := strings.Trim(*body.APIKey, `"`)
		if err := h.creds.Save(r.Context(), session.UserID, body.Provider, key); err != nil {
			slog.Error("save credential failed", slog.Any("err", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
