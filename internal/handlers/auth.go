package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

// bcryptCost is the work factor for hashing passwords. Overridden to bcrypt.MinCost in tests.
var bcryptCost = bcrypt.DefaultCost

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.users.GetUserByUsername(r.Context(), body.Username)
	if err != nil {
		writeUnauthorized(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeUnauthorized(w)
		return
	}

	session, err := h.sessions.CreateSession(r.Context(), user.ID, time.Now().Add(30*24*time.Hour))
	if err != nil {
		slog.Error("create session failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, newSessionCookie(session.ID, 30*24*60*60))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"username": user.Username})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if ok {
		if err := h.sessions.DeleteSession(r.Context(), session.ID); err != nil {
			slog.Error("delete session failed", slog.Any("err", err))
		}
	}
	http.SetCookie(w, newSessionCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":       session.UserID,
		"username": session.Username,
	})
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Username == "" || body.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		slog.Error("bcrypt failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.users.CreateUser(r.Context(), body.Username, string(hash))
	if err != nil {
		if errors.Is(err, providers.ErrUsernameTaken) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"username already taken"}`))
			return
		}
		slog.Error("create user failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session, err := h.sessions.CreateSession(r.Context(), user.ID, time.Now().Add(30*24*time.Hour))
	if err != nil {
		slog.Error("create session after signup failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, newSessionCookie(session.ID, 30*24*60*60))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"username": user.Username})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
