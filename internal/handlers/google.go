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

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
)

const oauthStateCookie = "oauth_state"

type GoogleHandler struct {
	client *igoogle.Client
}

func NewGoogleHandler(client *igoogle.Client) *GoogleHandler {
	return &GoogleHandler{client: client}
}

func (h *GoogleHandler) OAuthStart(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		WriteError(w, r, err)
		return
	}
	setStateCookie(w, state)
	http.Redirect(w, r, h.client.AuthURL(state), http.StatusTemporaryRedirect)
}

func (h *GoogleHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if !validateStateCookie(r, r.URL.Query().Get("state")) {
		WriteError(w, r, apperr.Invalid("invalid oauth state"))
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
		WriteError(w, r, apperr.Invalid("missing code"))
		return
	}

	tok, err := h.client.Exchange(r.Context(), code)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	if err := h.client.SaveToken(r.Context(), session.UserID, tok); err != nil {
		WriteError(w, r, err)
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
		switch {
		case errors.Is(err, providers.ErrGoogleTokenNotFound):
			WriteJSON(w, http.StatusOK, response{Connected: false})
		case errors.Is(err, providers.ErrGoogleTokenUnusable):
			slog.Warn("google token unusable, treating as disconnected", slog.Any("err", err))
			WriteJSON(w, http.StatusOK, response{Connected: false})
		default:
			WriteError(w, r, err)
		}
		return
	}

	resp, err := hc.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		WriteError(w, r, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, http.StatusOK, response{Connected: true, Email: info.Email})
}

func (h *GoogleHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return
	}

	if err := h.client.DeleteToken(r.Context(), session.UserID); err != nil {
		WriteError(w, r, err)
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
