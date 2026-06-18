package scraper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestAPIPublisher_CorrectRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotAuth string
	var gotBody dto.Job

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pub := NewAPIPublisher(srv.URL, "test-token")
	job := dto.Job{Title: "Engineer", URL: "https://example.com/job/1"}

	if err := pub.Publish(context.Background(), []dto.Job{job}); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/ingest" {
		t.Errorf("path = %q, want /ingest", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", gotAuth)
	}
	if gotBody.Title != job.Title {
		t.Errorf("body.Title = %q, want %q", gotBody.Title, job.Title)
	}
}

func TestAPIPublisher_2xx_ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	pub := NewAPIPublisher(srv.URL, "tok")
	if err := pub.Publish(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}}); err != nil {
		t.Errorf("expected nil error on 201, got %v", err)
	}
}

func TestAPIPublisher_4xx_ReturnsError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	pub := NewAPIPublisher(srv.URL, "tok")
	err := pub.Publish(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}})
	if err == nil {
		t.Error("expected error on 400, got nil")
	}
	if calls != 1 {
		t.Errorf("4xx should not be retried: got %d calls, want 1", calls)
	}
}

func TestAPIPublisher_5xx_RetriesAndReturnsError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	pub := NewAPIPublisher(srv.URL, "tok")
	pub.initialBackoff = 0 // no sleep between retries in tests

	err := pub.Publish(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}})
	if err == nil {
		t.Error("expected error after 5xx retries, got nil")
	}
	// initial attempt + 2 retries = 3 total
	if calls != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}
