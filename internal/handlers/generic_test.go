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

func TestHandle(t *testing.T) {
	tests := []struct {
		name       string
		decode     func(*http.Request) (string, error)
		call       func(context.Context, string) (string, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "decode error goes to writeError, call is never reached",
			decode:     func(*http.Request) (string, error) { return "", apperr.Invalid("bad request") },
			call:       func(context.Context, string) (string, error) { t.Fatal("call should not run"); return "", nil },
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad request",
		},
		{
			name:       "call error goes to writeError",
			decode:     func(*http.Request) (string, error) { return "in", nil },
			call:       func(context.Context, string) (string, error) { return "", errors.New("boom") },
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "decoded request reaches call, call's result reaches respond",
			decode:     func(*http.Request) (string, error) { return "in", nil },
			call:       func(_ context.Context, in string) (string, error) { return in + ":out", nil },
			wantStatus: http.StatusOK,
			wantBody:   "in:out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handle(tt.decode, tt.call, respondJSON[string](http.StatusOK))
			w := httptest.NewRecorder()
			h(w, httptest.NewRequest(http.MethodGet, "/x", nil))
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestQueryDecodesURLParamsIntoDto(t *testing.T) {
	type q struct {
		StatusID string `json:"status_id"`
	}
	h := Query(func(_ context.Context, userID string, got q) (string, error) {
		return userID + ":" + got.StatusID, nil
	})
	req := withSession(httptest.NewRequest(http.MethodGet, "/x?status_id=s1", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "user-1:s1") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestQueryMissingParamDecodesToZeroValue(t *testing.T) {
	type q struct {
		StatusID string `json:"status_id"`
	}
	h := Query(func(_ context.Context, _ string, got q) (string, error) {
		return got.StatusID, nil
	})
	req := withSession(httptest.NewRequest(http.MethodGet, "/x", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

func TestCreateFillsMultiplePathParams(t *testing.T) {
	type in struct {
		DocID string `json:"-" path:"docId"`
		TabID string `json:"-" path:"tabId"`
	}
	var got in
	h := Create(func(_ context.Context, _ string, body in) (struct{}, error) {
		got = body
		return struct{}{}, nil
	})
	req := withRouteID(withSession(httptest.NewRequest(http.MethodPost, "/x/d1/tabs/t1/hide", nil), "user-1"), "docId", "d1", "tabId", "t1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body.String())
	}
	if got.DocID != "d1" || got.TabID != "t1" {
		t.Fatalf("in = %+v, want DocID=d1 TabID=t1", got)
	}
}

func TestCreatePathTagCannotBeOverriddenByBody(t *testing.T) {
	type in struct {
		ID   string `json:"-" path:"id"`
		Name string `json:"name"`
	}
	var got in
	h := Create(func(_ context.Context, _ string, body in) (struct{}, error) {
		got = body
		return struct{}{}, nil
	})
	req := withRouteID(withSession(httptest.NewRequest(http.MethodPost, "/x/real-id", strings.NewReader(`{"id":"spoofed","name":"a"}`)), "user-1"), "id", "real-id")
	w := httptest.NewRecorder()
	h(w, req)
	if got.ID != "real-id" {
		t.Fatalf("ID = %q, want it to come from the path, not the body", got.ID)
	}
}
