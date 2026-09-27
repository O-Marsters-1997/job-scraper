package identity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

type fakeSessionGetter struct {
	sessions map[string]dto.Session
	getErr   error
}

func (f *fakeSessionGetter) GetSession(_ context.Context, id string) (dto.Session, error) {
	if f.getErr != nil {
		return dto.Session{}, f.getErr
	}
	s, ok := f.sessions[id]
	if !ok {
		return dto.Session{}, errors.New("not found")
	}
	return s, nil
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := handlers.Session(r)
	if !ok {
		http.Error(w, "no session in context", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Username", session.Username)
	w.WriteHeader(http.StatusOK)
}

func TestSessionMiddlewareNoCookie(t *testing.T) {
	mw := sessionMiddleware(&fakeSessionGetter{sessions: map[string]dto.Session{}})
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestSessionMiddlewareUnknownSession(t *testing.T) {
	mw := sessionMiddleware(&fakeSessionGetter{getErr: errors.New("not found")})
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "unknown-id"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestSessionMiddlewareValidSession(t *testing.T) {
	sg := &fakeSessionGetter{sessions: map[string]dto.Session{
		"valid-session": {ID: "valid-session", UserID: "user-1", Username: "alice", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	mw := sessionMiddleware(sg)
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
