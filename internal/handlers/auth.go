package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

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

func Login(svc authSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeBody[struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}](w, r)
		if !ok {
			return
		}
		session, user, err := svc.Login(r.Context(), body.Username, body.Password)
		if err != nil {
			writeError(w, r, err)
			return
		}
		http.SetCookie(w, newSessionCookie(session.ID, 30*24*60*60))
		writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
	}
}

func Signup(svc authSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeBody[struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}](w, r)
		if !ok {
			return
		}
		session, user, err := svc.Signup(r.Context(), body.Username, body.Password, body.Email)
		if err != nil {
			writeError(w, r, err)
			return
		}
		http.SetCookie(w, newSessionCookie(session.ID, 30*24*60*60))
		writeJSON(w, http.StatusCreated, map[string]string{"username": user.Username})
	}
}

func Logout(svc authSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := auth.SessionFromContext(r.Context())
		if ok {
			_ = svc.Logout(r.Context(), session.ID)
		}
		http.SetCookie(w, newSessionCookie("", -1))
		w.WriteHeader(http.StatusNoContent)
	}
}

func Me(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{
		"id":       session.UserID,
		"username": session.Username,
	})
}
