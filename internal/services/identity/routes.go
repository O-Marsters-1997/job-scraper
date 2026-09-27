package identity

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

// PublicRoutes mounts login and signup outside the session-protected group.
func (m *Module) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", loginHandler(m.service))
	r.Post("/auth/signup", signupHandler(m.service))
}

// Routes mounts logout and me inside the session-protected group.
func (m *Module) Routes(r chi.Router) {
	r.Post("/auth/logout", logoutHandler(m.service))
	r.Get("/auth/me", meHandler)
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
		handlers.WriteJSON(w, status, dto.AuthUserView{Username: out.user.Username})
	}
}

func loginHandler(svc *Service) http.HandlerFunc {
	return handlers.Handle(
		handlers.DecodeBody[dto.LoginInput],
		func(ctx context.Context, in dto.LoginInput) (sessionUser, error) {
			session, user, err := svc.Login(ctx, in.Username, in.Password)
			return sessionUser{session, user}, err
		},
		respondSession(http.StatusOK),
	)
}

func signupHandler(svc *Service) http.HandlerFunc {
	return handlers.Handle(
		handlers.DecodeBody[dto.SignupInput],
		func(ctx context.Context, in dto.SignupInput) (sessionUser, error) {
			session, user, err := svc.Signup(ctx, in.Username, in.Password, in.Email)
			return sessionUser{session, user}, err
		},
		respondSession(http.StatusCreated),
	)
}

func logoutHandler(svc *Service) http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (string, error) {
			session, _ := handlers.Session(r)
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

var meHandler = handlers.Handle(
	func(r *http.Request) (dto.MeView, error) {
		session, ok := handlers.Session(r)
		if !ok {
			return dto.MeView{}, apperr.Unauthorized("unauthorized")
		}
		return dto.MeView{ID: session.UserID, Username: session.Username}, nil
	},
	func(_ context.Context, v dto.MeView) (dto.MeView, error) { return v, nil },
	func(w http.ResponseWriter, _ *http.Request, res dto.MeView) {
		handlers.WriteJSON(w, http.StatusOK, res)
	},
)
