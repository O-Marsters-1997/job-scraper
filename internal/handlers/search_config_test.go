package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestSearchConfigGet_HappyPath(t *testing.T) {
	db := providers.NewMockSearchConfigProvider()
	db.Seed(dto.SearchConfig{
		UserID:          "user-1",
		Role:            "Software Engineer",
		Location:        "London",
		Keywords:        []string{"Go", "Kubernetes"},
		RelevanceCutoff: 70,
		NotifyThreshold: 80,
	})

	h := NewSearchConfigHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/search-config", nil)
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "s1", UserID: "user-1", Username: "alice",
	}))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("want application/json, got %s", ct)
	}
	var got dto.SearchConfig
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Role != "Software Engineer" {
		t.Errorf("want role %q, got %q", "Software Engineer", got.Role)
	}
	if len(got.Keywords) != 2 {
		t.Errorf("want 2 keywords, got %d", len(got.Keywords))
	}
}

func TestSearchConfigGet_Unauthenticated(t *testing.T) {
	h := NewSearchConfigHandler(providers.NewMockSearchConfigProvider())
	req := httptest.NewRequest(http.MethodGet, "/search-config", nil)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestSearchConfigGet_DBError(t *testing.T) {
	db := providers.NewMockSearchConfigProvider()
	db.GetErr = errors.New("db down")

	h := NewSearchConfigHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/search-config", nil)
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "s1", UserID: "user-1", Username: "alice",
	}))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestSearchConfigPut_HappyPath(t *testing.T) {
	db := providers.NewMockSearchConfigProvider()

	h := NewSearchConfigHandler(db)
	body, _ := json.Marshal(dto.SearchConfig{
		Role:            "Platform Engineer",
		Location:        "Remote",
		Keywords:        []string{"Go"},
		RelevanceCutoff: 65,
		NotifyThreshold: 75,
	})
	req := httptest.NewRequest(http.MethodPut, "/search-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "s1", UserID: "user-1", Username: "alice",
	}))
	w := httptest.NewRecorder()

	h.Put(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("want application/json, got %s", ct)
	}
	var got dto.SearchConfig
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Role != "Platform Engineer" {
		t.Errorf("want role %q, got %q", "Platform Engineer", got.Role)
	}
	if got.UserID != "user-1" {
		t.Errorf("want userID %q, got %q", "user-1", got.UserID)
	}
}

func TestSearchConfigPut_Unauthenticated(t *testing.T) {
	h := NewSearchConfigHandler(providers.NewMockSearchConfigProvider())
	body, _ := json.Marshal(dto.SearchConfig{Role: "Engineer"})
	req := httptest.NewRequest(http.MethodPut, "/search-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Put(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestSearchConfigPut_InvalidJSON(t *testing.T) {
	h := NewSearchConfigHandler(providers.NewMockSearchConfigProvider())
	req := httptest.NewRequest(http.MethodPut, "/search-config", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "s1", UserID: "user-1", Username: "alice",
	}))
	w := httptest.NewRecorder()

	h.Put(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestSearchConfigPut_DBError(t *testing.T) {
	db := providers.NewMockSearchConfigProvider()
	db.UpsertErr = errors.New("db down")

	h := NewSearchConfigHandler(db)
	body, _ := json.Marshal(dto.SearchConfig{Role: "Engineer"})
	req := httptest.NewRequest(http.MethodPut, "/search-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithSession(context.Background(), dto.Session{
		ID: "s1", UserID: "user-1", Username: "alice",
	}))
	w := httptest.NewRecorder()

	h.Put(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
