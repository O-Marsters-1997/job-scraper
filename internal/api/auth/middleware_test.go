package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/api/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := handlers.Session(r)
	if !ok {
		http.Error(w, "no session in context", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Username", session.Username)
	w.WriteHeader(http.StatusOK)
}

func TestMiddleware_NoCookie(t *testing.T) {
	sp := providers.NewMockSessionProvider()
	mw := auth.Middleware(sp)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestMiddleware_UnknownSession(t *testing.T) {
	sp := providers.NewMockSessionProvider()
	sp.GetErr = fmt.Errorf("not found")
	mw := auth.Middleware(sp)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "unknown-id"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestMiddleware_ExpiredSession(t *testing.T) {
	sp := providers.NewMockSessionProvider()
	sp.Seed(dto.Session{
		ID:        "expired-session",
		UserID:    "user-1",
		Username:  "alice",
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	mw := auth.Middleware(sp)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "expired-session"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestMiddleware_ValidSession(t *testing.T) {
	sp := providers.NewMockSessionProvider()
	sp.Seed(dto.Session{
		ID:        "valid-session",
		UserID:    "user-1",
		Username:  "alice",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	mw := auth.Middleware(sp)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-Username"); got != "alice" {
		t.Errorf("want username alice in context, got %q", got)
	}
}
