package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

func TestUserNoSessionReturns401(t *testing.T) {
	h := User(func(context.Context, string) (string, error) {
		t.Fatal("fn should not be called without a session")
		return "", nil
	}, http.StatusOK)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
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

func TestKindedErrorMapsToItsStatus(t *testing.T) {
	h := User(func(context.Context, string) (string, error) {
		return "", apperr.Conflict("board belongs to another company")
	}, http.StatusOK)
	req := withSession(httptest.NewRequest(http.MethodGet, "/x", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if !strings.Contains(w.Body.String(), "board belongs to another company") {
		t.Fatalf("body = %q, want the kinded message", w.Body.String())
	}
}

func TestUnkindedErrorHidesMessageAndReturns500(t *testing.T) {
	h := User(func(context.Context, string) (string, error) {
		return "", errors.New("pq: connection refused on 10.0.0.5:5432")
	}, http.StatusOK)
	req := withSession(httptest.NewRequest(http.MethodGet, "/x", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "10.0.0.5") {
		t.Fatalf("body leaked the underlying error: %q", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Fatalf("body = %q, want the generic message", w.Body.String())
	}
}

func TestNilSliceIsWrittenAsEmptyArray(t *testing.T) {
	h := User(func(context.Context, string) ([]string, error) {
		return nil, nil
	}, http.StatusOK)
	req := withSession(httptest.NewRequest(http.MethodGet, "/x", nil), "user-1")
	w := httptest.NewRecorder()
	h(w, req)
	if got := strings.TrimSpace(w.Body.String()); got != "[]" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestNoContentStatusWritesNoBody(t *testing.T) {
	h := ID(func(context.Context, string, string) (struct{}, error) {
		return struct{}{}, nil
	}, http.StatusNoContent)
	req := withSession(httptest.NewRequest(http.MethodDelete, "/x/1", nil), "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", w.Body.String())
	}
}

func TestIDPassesChiURLParam(t *testing.T) {
	var gotID string
	h := ID(func(_ context.Context, userID, id string) (string, error) {
		gotID = id
		return userID, nil
	}, http.StatusOK)
	req := withSession(httptest.NewRequest(http.MethodDelete, "/x/abc-123", nil), "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h(w, req)
	if gotID != "abc-123" {
		t.Fatalf("id = %q, want abc-123", gotID)
	}
}

func TestBodyIDCombinesIDAndDecodedBody(t *testing.T) {
	type in struct {
		Notes string `json:"notes"`
	}
	var gotID, gotUserID, gotNotes string
	h := BodyID(func(_ context.Context, userID, id string, body in) (string, error) {
		gotUserID, gotID, gotNotes = userID, id, body.Notes
		return "ok", nil
	}, http.StatusOK)
	req := withSession(httptest.NewRequest(http.MethodPatch, "/x/app-1", strings.NewReader(`{"notes":"followed up"}`)), "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "app-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if gotUserID != "user-1" || gotID != "app-1" || gotNotes != "followed up" {
		t.Fatalf("got userID=%q id=%q notes=%q", gotUserID, gotID, gotNotes)
	}
}
