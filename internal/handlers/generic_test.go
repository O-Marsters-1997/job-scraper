package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
