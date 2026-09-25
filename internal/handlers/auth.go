package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// authSvc is the interface handlers.Login/Signup/Logout need from
// services/auth.Service; declared here, not imported, so the handler
// package doesn't depend on the service package.
type authSvc interface {
	Login(ctx context.Context, username, password string) (dto.Session, dto.User, error)
	Signup(ctx context.Context, username, password, email string) (dto.Session, dto.User, error)
	Logout(ctx context.Context, sessionID string) error
}

func newSessionCookie(id string, maxAge int) *http.Cookie {
	secure := os.Getenv("COOKIE_SECURE") == "true"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	return &http.Cookie{
		Name:     "session_id",
		Value:    id,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
		Path:     "/",
		MaxAge:   maxAge,
	}
}

type sessionUser struct {
	session dto.Session
	user    dto.User
}

func respondSession(status int) func(http.ResponseWriter, *http.Request, sessionUser) {
	return func(w http.ResponseWriter, _ *http.Request, out sessionUser) {
		http.SetCookie(w, newSessionCookie(out.session.ID, 30*24*60*60))
		respond(w, status, dto.AuthUserView{Username: out.user.Username})
	}
}

// Login authenticates a user by username/password and starts a session.
func Login(svc authSvc) http.HandlerFunc {
	return Handle(
		decodeBody[dto.LoginInput],
		func(ctx context.Context, in dto.LoginInput) (sessionUser, error) {
			session, user, err := svc.Login(ctx, in.Username, in.Password)
			return sessionUser{session, user}, err
		},
		respondSession(http.StatusOK),
	)
}

// Signup creates a user and starts a session.
func Signup(svc authSvc) http.HandlerFunc {
	return Handle(
		decodeBody[dto.SignupInput],
		func(ctx context.Context, in dto.SignupInput) (sessionUser, error) {
			session, user, err := svc.Signup(ctx, in.Username, in.Password, in.Email)
			return sessionUser{session, user}, err
		},
		respondSession(http.StatusCreated),
	)
}

// Logout clears the caller's session; the service error is swallowed so the
// browser is always logged out even if the underlying session delete fails.
func Logout(svc authSvc) http.HandlerFunc {
	return Handle(
		func(r *http.Request) (string, error) {
			session, _ := auth.SessionFromContext(r.Context())
			return session.ID, nil
		},
		func(ctx context.Context, sessionID string) (struct{}, error) {
			if sessionID != "" {
				_ = svc.Logout(ctx, sessionID)
			}
			return struct{}{}, nil
		},
		func(w http.ResponseWriter, _ *http.Request, _ struct{}) {
			http.SetCookie(w, newSessionCookie("", -1))
			w.WriteHeader(http.StatusNoContent)
		},
	)
}

// Me returns the caller's session identity.
var Me = Handle(
	func(r *http.Request) (dto.MeView, error) {
		session, ok := auth.SessionFromContext(r.Context())
		if !ok {
			return dto.MeView{}, apperr.Unauthorized("unauthorized")
		}
		return dto.MeView{ID: session.UserID, Username: session.Username}, nil
	},
	pass[dto.MeView],
	respondJSON[dto.MeView](http.StatusOK),
)
