package identity

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
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

const oauthStateCookie = "oauth_state"

type googleAuthConnector interface {
	AuthURL(state string) string
	Connect(ctx context.Context, userID, code string) error
}

type oauthRedirect struct {
	state, authURL string
}

func oauthStartHandler(svc googleAuthConnector) http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (string, error) { return generateState() },
		func(_ context.Context, state string) (oauthRedirect, error) {
			return oauthRedirect{state: state, authURL: svc.AuthURL(state)}, nil
		},
		func(w http.ResponseWriter, r *http.Request, out oauthRedirect) {
			setOAuthStateCookie(w, out.state)
			http.Redirect(w, r, out.authURL, http.StatusTemporaryRedirect)
		},
	)
}

type oauthConnect struct {
	userID, code string
}

func oauthCallbackHandler(svc googleAuthConnector) http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (oauthConnect, error) {
			if !validOAuthStateCookie(r, r.URL.Query().Get("state")) {
				return oauthConnect{}, apperr.Invalid("invalid oauth state")
			}
			code := r.URL.Query().Get("code")
			if code == "" {
				return oauthConnect{}, apperr.Invalid("missing code")
			}
			uid, err := handlers.UserID(r)
			if err != nil {
				return oauthConnect{}, err
			}
			return oauthConnect{userID: uid, code: code}, nil
		},
		func(ctx context.Context, in oauthConnect) (struct{}, error) {
			return struct{}{}, svc.Connect(ctx, in.userID, in.code)
		},
		func(w http.ResponseWriter, r *http.Request, _ struct{}) {
			http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", MaxAge: -1, Path: "/"})
			http.Redirect(w, r, "/settings/integrations", http.StatusTemporaryRedirect)
		},
	)
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	signed := signOAuthState(state)
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

func validOAuthStateCookie(r *http.Request, state string) bool {
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" {
		return false
	}
	return hmac.Equal([]byte(cookie.Value), []byte(signOAuthState(state)))
}

func signOAuthState(state string) string {
	secret := os.Getenv("SESSION_SECRET")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(state))
	return fmt.Sprintf("%s:%s", state, hex.EncodeToString(mac.Sum(nil)))
}
