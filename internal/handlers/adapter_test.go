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

func TestUser(t *testing.T) {
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
			handler: User(func(context.Context, string) (string, error) {
				return "unexpected", nil
			}, http.StatusOK),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "kinded error maps to its status",
			session: true,
			handler: User(func(context.Context, string) (string, error) {
				return "", apperr.Conflict("board belongs to another company")
			}, http.StatusOK),
			wantStatus:  http.StatusConflict,
			wantBodyHas: "board belongs to another company",
		},
		{
			name:    "unkinded error hides message and returns 500",
			session: true,
			handler: User(func(context.Context, string) (string, error) {
				return "", errors.New("pq: connection refused on 10.0.0.5:5432")
			}, http.StatusOK),
			wantStatus:    http.StatusInternalServerError,
			wantBodyHas:   "internal server error",
			wantBodyLacks: "10.0.0.5",
		},
		{
			name:    "nil slice is written as empty array",
			session: true,
			handler: User(func(context.Context, string) ([]string, error) {
				return nil, nil
			}, http.StatusOK),
			wantStatus:  http.StatusOK,
			wantBodyHas: "[]",
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

func TestBodyDecodeErrorReturns400WithJSONBody(t *testing.T) {
	h := Body(func(context.Context, string, struct{ Name string }) (string, error) {
		t.Fatal("fn should not be called on decode failure")
		return "", nil
	}, http.StatusOK)
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

func TestID(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantStatus  int
		wantBodyHas string
	}{
		{
			name: "no content status writes no body",
			handler: ID(func(context.Context, string, string) (struct{}, error) {
				return struct{}{}, nil
			}, http.StatusNoContent),
			wantStatus: http.StatusNoContent,
		},
		{
			name: "passes chi url param as id",
			handler: ID(func(_ context.Context, userID, id string) (string, error) {
				return userID + ":" + id, nil
			}, http.StatusOK),
			wantStatus:  http.StatusOK,
			wantBodyHas: "user-1:abc-123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := withRouteID(withSession(httptest.NewRequest(http.MethodDelete, "/x/abc-123", nil), "user-1"), "abc-123")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBodyHas != "" && !strings.Contains(w.Body.String(), tt.wantBodyHas) {
				t.Fatalf("body = %q, want it to contain %q", w.Body.String(), tt.wantBodyHas)
			}
			if tt.wantStatus == http.StatusNoContent && w.Body.Len() != 0 {
				t.Fatalf("body = %q, want empty", w.Body.String())
			}
		})
	}
}

func TestBodyIDCombinesIDAndDecodedBody(t *testing.T) {
	type in struct {
		Notes string `json:"notes"`
	}
	h := BodyID(func(_ context.Context, userID, id string, body in) (string, error) {
		return userID + ":" + id + ":" + body.Notes, nil
	}, http.StatusOK)
	req := withRouteID(withSession(httptest.NewRequest(http.MethodPatch, "/x/app-1", strings.NewReader(`{"notes":"followed up"}`)), "user-1"), "app-1")
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
