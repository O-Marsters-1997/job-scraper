package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

const oauthStateCookie = "oauth_state"

// googleSvc is the interface handlers.OAuthStart/OAuthCallback need from
// services/google.Service; declared here, not imported.
type googleSvc interface {
	AuthURL(state string) string
	Connect(ctx context.Context, userID, code string) error
}

// OAuthStart is public so the OAuth redirect URL stays clean; it redirects,
// so it stays a misfit rather than going through the generic adapter.
func OAuthStart(svc googleSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := generateState()
		if err != nil {
			writeError(w, r, err)
			return
		}
		setStateCookie(w, state)
		http.Redirect(w, r, svc.AuthURL(state), http.StatusTemporaryRedirect)
	}
}

// OAuthCallback redirects, so it stays a misfit rather than going through
// the generic adapter.
func OAuthCallback(svc googleSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validateStateCookie(r, r.URL.Query().Get("state")) {
			writeError(w, r, apperr.Invalid("invalid oauth state"))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", MaxAge: -1, Path: "/"})

		code := r.URL.Query().Get("code")
		if code == "" {
			writeError(w, r, apperr.Invalid("missing code"))
			return
		}

		userID, ok := caller(w, r)
		if !ok {
			return
		}
		if err := svc.Connect(r.Context(), userID, code); err != nil {
			writeError(w, r, err)
			return
		}
		http.Redirect(w, r, "/settings/integrations", http.StatusTemporaryRedirect)
	}
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
