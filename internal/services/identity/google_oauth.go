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
	"regexp"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

const (
	oauthStateCookie  = "oauth_state"
	oauthReturnCookie = "oauth_return"
	defaultReturnPath = "/settings/integrations"
)

var tailorReturnPath = regexp.MustCompile(`^/jobs/[A-Za-z0-9_-]+/tailor$`)

func safeReturnPath(p string) string {
	if p == defaultReturnPath || tailorReturnPath.MatchString(p) {
		return p
	}
	return defaultReturnPath
}

type googleAuthConnector interface {
	AuthURL(state string, write bool) string
	Connect(ctx context.Context, userID, code string) error
}

type oauthStart struct {
	state      string
	write      bool
	returnPath string
}

type oauthRedirect struct {
	state, authURL, returnPath string
}

func oauthStartHandler(svc googleAuthConnector) http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (oauthStart, error) {
			state, err := generateState()
			if err != nil {
				return oauthStart{}, err
			}
			q := r.URL.Query()
			return oauthStart{
				state:      state,
				write:      q.Get("write") == "1",
				returnPath: safeReturnPath(q.Get("return")),
			}, nil
		},
		func(_ context.Context, in oauthStart) (oauthRedirect, error) {
			return oauthRedirect{
				state:      in.state,
				authURL:    svc.AuthURL(in.state, in.write),
				returnPath: in.returnPath,
			}, nil
		},
		func(w http.ResponseWriter, r *http.Request, out oauthRedirect) {
			setOAuthStateCookie(w, out.state)
			setOAuthReturnCookie(w, out.returnPath)
			http.Redirect(w, r, out.authURL, http.StatusTemporaryRedirect)
		},
	)
}

type oauthConnect struct {
	userID, code, returnPath string
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
			returnPath := defaultReturnPath
			if c, err := r.Cookie(oauthReturnCookie); err == nil {
				returnPath = safeReturnPath(c.Value)
			}
			return oauthConnect{userID: uid, code: code, returnPath: returnPath}, nil
		},
		func(ctx context.Context, in oauthConnect) (string, error) {
			return in.returnPath, svc.Connect(ctx, in.userID, in.code)
		},
		func(w http.ResponseWriter, r *http.Request, returnPath string) {
			http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", MaxAge: -1, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: oauthReturnCookie, Value: "", MaxAge: -1, Path: "/"})
			http.Redirect(w, r, returnPath, http.StatusTemporaryRedirect)
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
	setOAuthCookie(w, oauthStateCookie, signOAuthState(state))
}

func setOAuthReturnCookie(w http.ResponseWriter, returnPath string) {
	setOAuthCookie(w, oauthReturnCookie, returnPath)
}

func setOAuthCookie(w http.ResponseWriter, name, value string) {
	secure := os.Getenv("COOKIE_SECURE") == "true"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
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
