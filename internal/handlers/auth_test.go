package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestLogin_ValidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	users := providers.NewMockUserProvider()
	users.Seed(dto.User{ID: "user-1", Username: "alice", PasswordHash: string(hash)})
	sessions := providers.NewMockSessionProvider()

	h := New(nil, users, sessions)
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("expected session_id cookie to be set")
	}
	if !cookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	users := providers.NewMockUserProvider()
	users.Seed(dto.User{ID: "user-1", Username: "alice", PasswordHash: string(hash)})
	sessions := providers.NewMockSessionProvider()

	h := New(nil, users, sessions)
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	users := providers.NewMockUserProvider()
	sessions := providers.NewMockSessionProvider()

	h := New(nil, users, sessions)
	body, _ := json.Marshal(map[string]string{"username": "nobody", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestLogout_ClearsSessionAndCookie(t *testing.T) {
	sessions := providers.NewMockSessionProvider()
	sessions.Seed(dto.Session{
		ID:        "session-abc",
		UserID:    "user-1",
		Username:  "alice",
		ExpiresAt: time.Now().Add(time.Hour),
	})

	h := New(nil, nil, sessions)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "session-abc", UserID: "user-1", Username: "alice",
	}))

	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}

	var found *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" {
			found = c
		}
	}
	if found == nil {
		t.Fatal("expected session_id cookie in response (to clear it)")
	}
	if found.MaxAge != -1 {
		t.Errorf("want MaxAge -1 (delete cookie), got %d", found.MaxAge)
	}
}

func TestMe_ReturnsCurrentUser(t *testing.T) {
	h := New(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "s1", UserID: "u1", Username: "alice",
	}))
	w := httptest.NewRecorder()

	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["username"] != "alice" {
		t.Errorf("want username alice, got %q", got["username"])
	}
	if got["id"] != "u1" {
		t.Errorf("want id u1, got %q", got["id"])
	}
}
