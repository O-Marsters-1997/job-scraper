package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

func TestGetAll(t *testing.T) {
	tests := []struct {
		name          string
		session       bool
		handler       http.HandlerFunc
		wantStatus    int
		wantBodyHas   string
		wantBodyLacks string
	}{
		{
			name: "no session returns 401",
			handler: GetAll(func(context.Context, string) (string, error) {
				return "unexpected", nil
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "kinded error maps to its status",
			session: true,
			handler: GetAll(func(context.Context, string) (string, error) {
				return "", apperr.Conflict("board belongs to another company")
			}),
			wantStatus:  http.StatusConflict,
			wantBodyHas: "board belongs to another company",
		},
		{
			name:    "unkinded error hides message and returns 500",
			session: true,
			handler: GetAll(func(context.Context, string) (string, error) {
				return "", errors.New("pq: connection refused on 10.0.0.5:5432")
			}),
			wantStatus:    http.StatusInternalServerError,
			wantBodyHas:   "internal server error",
			wantBodyLacks: "10.0.0.5",
		},
		{
			name:    "nil slice is written as empty array",
			session: true,
			handler: GetAll(func(context.Context, string) ([]string, error) {
				return nil, nil
			}),
			wantStatus:  http.StatusOK,
			wantBodyHas: "[]",
		},
		{
			name:    "fields attached to an error are merged into the body",
			session: true,
			handler: GetAll(func(context.Context, string) (string, error) {
				return "", apperr.WithFields(apperr.Conflict("status is in use"), map[string]any{"count": float64(3)})
			}),
			wantStatus:  http.StatusConflict,
			wantBodyHas: `"count":3`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.session {
				req = withSession(req, "user-1")
			}
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBodyHas != "" && !strings.Contains(w.Body.String(), tt.wantBodyHas) {
				t.Fatalf("body = %q, want it to contain %q", w.Body.String(), tt.wantBodyHas)
			}
			if tt.wantBodyLacks != "" && strings.Contains(w.Body.String(), tt.wantBodyLacks) {
				t.Fatalf("body = %q, want it not to contain %q", w.Body.String(), tt.wantBodyLacks)
			}
		})
	}
}

func TestCreateDecodeErrorReturns400WithJSONBody(t *testing.T) {
	h := Create(func(context.Context, string, struct{ Name string }) (string, error) {
		t.Fatal("fn should not be called on decode failure")
		return "", nil
	})
	req := withSession(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("not json")), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if !strings.Contains(w.Body.String(), `"error"`) {
		t.Fatalf("body = %q, want an error field", w.Body.String())
	}
}

func TestCreateEmptyBodyDecodesToZeroValue(t *testing.T) {
	h := Create(func(_ context.Context, userID string, in struct{ Name string }) (string, error) {
		return userID + ":" + in.Name, nil
	})
	req := withSession(httptest.NewRequest(http.MethodPost, "/x", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "user-1:") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestGetByID(t *testing.T) {
	req := withRouteID(withSession(httptest.NewRequest(http.MethodGet, "/x/abc-123", nil), "user-1"), "id", "abc-123")
	w := httptest.NewRecorder()
	GetByID(func(_ context.Context, userID, id string) (string, error) {
		return userID + ":" + id, nil
	})(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "user-1:abc-123") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestUpdateCombinesPathAndDecodedBody(t *testing.T) {
	type in struct {
		ID    string `json:"-" path:"id"`
		Notes string `json:"notes"`
	}
	h := Update(func(_ context.Context, userID string, body in) (string, error) {
		return userID + ":" + body.ID + ":" + body.Notes, nil
	})
	req := withRouteID(withSession(httptest.NewRequest(http.MethodPatch, "/x/app-1", strings.NewReader(`{"notes":"followed up"}`)), "user-1"), "id", "app-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	want := "user-1:app-1:followed up"
	if got := w.Body.String(); !strings.Contains(got, want) {
		t.Fatalf("body = %q, want it to contain %q", got, want)
	}
}

func TestUpdateVoidOutputWrites204(t *testing.T) {
	h := Update(func(context.Context, string, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
	req := withSession(httptest.NewRequest(http.MethodPut, "/x", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", w.Body.String())
	}
}

func TestDelete(t *testing.T) {
	req := withRouteID(withSession(httptest.NewRequest(http.MethodDelete, "/x/abc-123", nil), "user-1"), "id", "abc-123")
	w := httptest.NewRecorder()
	var gotID string
	Delete(func(_ context.Context, _, id string) error {
		gotID = id
		return nil
	})(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if gotID != "abc-123" {
		t.Fatalf("id = %q, want abc-123", gotID)
	}
}
