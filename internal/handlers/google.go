package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
)

const oauthStateCookie = "oauth_state"

type GoogleHandler struct {
	client   *igoogle.Client
	sessions providers.SessionProvider
}

func NewGoogleHandler(client *igoogle.Client, sessions providers.SessionProvider) *GoogleHandler {
	return &GoogleHandler{client: client, sessions: sessions}
}

func (h *GoogleHandler) OAuthStart(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		slog.Error("generate oauth state failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	setStateCookie(w, state)
	http.Redirect(w, r, h.client.AuthURL(state), http.StatusTemporaryRedirect)
}

func (h *GoogleHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if !validateStateCookie(r, r.URL.Query().Get("state")) {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   oauthStateCookie,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	tok, err := h.client.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("oauth token exchange failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	if err := h.client.SaveToken(r.Context(), session.UserID, tok); err != nil {
		slog.Error("save google token failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings/integrations", http.StatusTemporaryRedirect)
}

func (h *GoogleHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	type response struct {
		Connected bool   `json:"connected"`
		Email     string `json:"email,omitempty"`
	}

	hc, err := h.client.HTTPClientForUser(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, providers.ErrGoogleTokenNotFound) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response{Connected: false})
			return
		}
		slog.Error("get google http client failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp, err := hc.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		slog.Error("google userinfo fetch failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		slog.Error("decode google userinfo failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{Connected: true, Email: info.Email})
}

func (h *GoogleHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	if err := h.client.DeleteToken(r.Context(), session.UserID); err != nil {
		slog.Error("delete google token failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// generateState returns a 16-byte cryptographically random hex string.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// setStateCookie writes an HMAC-signed state cookie to the response.
func setStateCookie(w http.ResponseWriter, state string) {
	signed := signState(state)
	secure := os.Getenv("COOKIE_SECURE") == "true"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    signed,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
		Path:     "/",
		MaxAge:   int((10 * time.Minute).Seconds()),
	})
}

func validateStateCookie(r *http.Request, state string) bool {
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" {
		return false
	}
	return hmac.Equal([]byte(cookie.Value), []byte(signState(state)))
}

// signState returns "state:HMAC(state)" using SESSION_SECRET as the key.
func signState(state string) string {
	secret := os.Getenv("SESSION_SECRET")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(state))
	return fmt.Sprintf("%s:%s", state, hex.EncodeToString(mac.Sum(nil)))
}
