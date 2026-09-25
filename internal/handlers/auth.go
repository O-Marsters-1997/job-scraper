package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

// bcryptCost is the work factor for hashing passwords. Overridden to bcrypt.MinCost in tests.
var bcryptCost = bcrypt.DefaultCost

type authStore interface {
	providers.UserProvider
	providers.SessionProvider
	SeedDefaultStatuses(ctx context.Context, userID string) error
}

type AuthHandler struct {
	store authStore
}

func NewAuthHandler(store authStore) *AuthHandler {
	return &AuthHandler{store: store}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, ok := DecodeJSON[struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}](w, r)
	if !ok {
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), body.Username)
	if err != nil {
		auth.WriteUnauthorized(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		auth.WriteUnauthorized(w)
		return
	}

	session, err := h.store.CreateSession(r.Context(), user.ID, time.Now().Add(30*24*time.Hour))
	if err != nil {
		WriteError(w, r, err)
		return
	}

	http.SetCookie(w, newSessionCookie(session.ID, 30*24*60*60))
	WriteJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if ok {
		if err := h.store.DeleteSession(r.Context(), session.ID); err != nil {
			slog.Error("delete session failed",
				slog.Any("err", err),
			)
		}
	}
	http.SetCookie(w, newSessionCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	WriteJSON(w, http.StatusOK, map[string]string{
		"id":       session.UserID,
		"username": session.Username,
	})
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	body, ok := DecodeJSON[struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}](w, r)
	if !ok {
		return
	}
	if body.Username == "" || body.Password == "" {
		WriteError(w, r, apperr.Invalid("bad request"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	user, err := h.store.CreateUser(r.Context(), body.Username, string(hash), body.Email)
	if err != nil {
		if errors.Is(err, providers.ErrUsernameTaken) {
			WriteError(w, r, apperr.Conflict(err.Error()))
			return
		}
		WriteError(w, r, err)
		return
	}

	if err := h.store.SeedDefaultStatuses(r.Context(), user.ID); err != nil {
		slog.Error("seed default statuses failed",
			slog.Any("err", err),
		)
	}

	session, err := h.store.CreateSession(r.Context(), user.ID, time.Now().Add(30*24*time.Hour))
	if err != nil {
		WriteError(w, r, err)
		return
	}

	http.SetCookie(w, newSessionCookie(session.ID, 30*24*60*60))
	WriteJSON(w, http.StatusCreated, map[string]string{"username": user.Username})
}
