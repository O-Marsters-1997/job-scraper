package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestProfileHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setup      func(*providers.MockProfileProvider)
		wantStatus int
		wantBody   map[string]string
	}{
		{
			name:   "returns username and email",
			userID: "user-1",
			setup: func(m *providers.MockProfileProvider) {
				m.Seed("user-1", dto.Profile{Username: "alice", Email: "alice@example.com"})
			},
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"username": "alice", "email": "alice@example.com"},
		},
		{
			name:   "returns username with empty email when not set",
			userID: "user-2",
			setup: func(m *providers.MockProfileProvider) {
				m.Seed("user-2", dto.Profile{Username: "bob", Email: ""})
			},
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"username": "bob", "email": ""},
		},
		{
			name:   "provider error returns 500",
			userID: "user-3",
			setup: func(m *providers.MockProfileProvider) {
				m.GetErr = errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockProfileProvider()
			if tt.setup != nil {
				tt.setup(store)
			}
			h := NewProfileHandler(store)

			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			req = withSession(req, tt.userID)
			w := httptest.NewRecorder()

			h.Get(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantBody == nil {
				return
			}
			var got map[string]string
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			for k, want := range tt.wantBody {
				if got[k] != want {
					t.Errorf("%s = %q; want %q", k, got[k], want)
				}
			}
		})
	}
}

func TestProfileHandler_Put(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		setup      func(*providers.MockProfileProvider)
		wantStatus int
	}{
		{
			name:       "updates email returns 204",
			body:       `{"email":"new@example.com"}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "clears email with empty string",
			body:       `{"email":""}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "bad JSON returns 400",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "provider error returns 500",
			body: `{"email":"x@example.com"}`,
			setup: func(m *providers.MockProfileProvider) {
				m.UpdateErr = errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockProfileProvider()
			store.Seed("user-1", dto.Profile{Username: "alice"})
			if tt.setup != nil {
				tt.setup(store)
			}
			h := NewProfileHandler(store)

			req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.Put(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
