package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func withSession(r *http.Request, userID string) *http.Request {
	ctx := auth.WithSession(r.Context(), dto.Session{UserID: userID})
	return r.WithContext(ctx)
}

func withRouteID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSourceTargetHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*providers.MockSourceTargetProvider)
		wantStatus int
		wantCount  int
	}{
		{
			name: "returns targets for current user",
			setup: func(s *providers.MockSourceTargetProvider) {
				_, _ = s.CreateSourceTarget(context.Background(), "user-1", "greenhouse", "acme", true, nil)
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "returns empty list when no targets",
			setup:      func(*providers.MockSourceTargetProvider) {},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSourceTargetProvider()
			tt.setup(store)
			h := NewSourceTargetHandler(store, nil)

			req := httptest.NewRequest(http.MethodGet, "/source-targets", nil)
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.List(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("want %d, got %d", tt.wantStatus, w.Code)
			}
			var got []dto.SourceTarget
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("want %d targets, got %d", tt.wantCount, len(got))
			}
		})
	}
}

func TestSourceTargetHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setup      func(*providers.MockSourceTargetProvider)
		wantStatus int
	}{
		{
			name:       "creates target successfully",
			body:       map[string]string{"source": "greenhouse", "value": "acme"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "rejects unknown source",
			body:       map[string]string{"source": "unknown-ats", "value": "something"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects bad URL for URL source",
			body:       map[string]string{"source": "linkedin", "value": "notaurl"},
			wantStatus: http.StatusBadRequest,
		},
		{
			// ErrDuplicateSourceTarget is not a pgconn.PgError so the 409 branch is
			// not reachable from the mock; the pgconn 23505 path is covered by DB
			// integration tests.
			name: "returns 500 on duplicate from mock",
			body: map[string]string{"source": "greenhouse", "value": "acme"},
			setup: func(s *providers.MockSourceTargetProvider) {
				s.CreateErr = providers.ErrDuplicateSourceTarget
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSourceTargetProvider()
			if tt.setup != nil {
				tt.setup(store)
			}
			h := NewSourceTargetHandler(store, nil)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/source-targets", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("want %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestSourceTargetHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		targetID   func(*providers.MockSourceTargetProvider) string
		wantStatus int
	}{
		{
			name: "updates enabled flag",
			targetID: func(s *providers.MockSourceTargetProvider) string {
				created, _ := s.CreateSourceTarget(context.Background(), "user-1", "greenhouse", "acme", true, nil)
				return created.ID
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns 404 for non-existent target",
			targetID:   func(*providers.MockSourceTargetProvider) string { return "non-existent-id" },
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSourceTargetProvider()
			id := tt.targetID(store)
			h := NewSourceTargetHandler(store, nil)

			body, _ := json.Marshal(map[string]bool{"enabled": false})
			req := httptest.NewRequest(http.MethodPatch, "/source-targets/"+id, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			req = withRouteID(req, id)
			w := httptest.NewRecorder()

			h.Update(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("want %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestSourceTargetHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		wantStatus int
	}{
		{
			name:       "deletes own target",
			userID:     "user-1",
			wantStatus: http.StatusNoContent,
		},
		{
			// Delete does not check ErrNotFound — callers must not learn whether
			// another user's target exists.
			name:       "returns 500 for another user's target",
			userID:     "user-2",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSourceTargetProvider()
			created, _ := store.CreateSourceTarget(context.Background(), "user-1", "greenhouse", "acme", true, nil)
			h := NewSourceTargetHandler(store, nil)

			req := httptest.NewRequest(http.MethodDelete, "/source-targets/"+created.ID, nil)
			req = withSession(req, tt.userID)
			req = withRouteID(req, created.ID)
			w := httptest.NewRecorder()

			h.Delete(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("want %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
