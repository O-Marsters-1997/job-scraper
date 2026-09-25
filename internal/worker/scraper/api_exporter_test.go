package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestAPIExporter_CorrectRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotAuth string
	var gotBody struct {
		Jobs []dto.Job `json:"jobs"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"results":[{"status":"new","job_id":"job-1"}]}`))
	}))
	defer srv.Close()

	pub := NewAPIExporter(srv.URL, "test-token")
	job := dto.Job{Title: "Engineer", URL: "https://example.com/job/1"}

	if err := pub.BulkExport(context.Background(), []dto.Job{job}); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/ingest/batch" {
		t.Errorf("path = %q, want /ingest/batch", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", gotAuth)
	}
	if len(gotBody.Jobs) != 1 || gotBody.Jobs[0].Title != job.Title {
		t.Errorf("body.Jobs = %+v, want one job titled %q", gotBody.Jobs, job.Title)
	}
}

func TestAPIExporter_2xx_ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"results":[{"status":"unchanged","job_id":"job-1"}]}`))
	}))
	defer srv.Close()

	pub := NewAPIExporter(srv.URL, "tok")
	if err := pub.BulkExport(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}}); err != nil {
		t.Errorf("expected nil error on 201, got %v", err)
	}
}

func TestAPIExporter_4xx_ReturnsError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	pub := NewAPIExporter(srv.URL, "tok")
	err := pub.BulkExport(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}})
	if err == nil {
		t.Error("expected error on 400, got nil")
	}
	if calls != 1 {
		t.Errorf("4xx should not be retried: got %d calls, want 1", calls)
	}
}

func TestAPIExporter_5xx_RetriesAndReturnsError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	pub := NewAPIExporter(srv.URL, "tok")
	pub.initialBackoff = 0

	err := pub.BulkExport(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}})
	if err == nil {
		t.Error("expected error after 5xx retries, got nil")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}

func TestAPIExporter_RetriesAmbiguousBatchWithSameIdentity(t *testing.T) {
	var firstBody []byte
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		calls++
		if calls == 1 {
			firstBody = body
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if !bytes.Equal(body, firstBody) {
			t.Errorf("retry body changed: first=%s second=%s", firstBody, body)
		}
		_, _ = w.Write([]byte(`{"results":[{"status":"unchanged","job_id":"job-1"}]}`))
	}))
	defer srv.Close()
	exporter := NewAPIExporter(srv.URL, "tok")
	exporter.initialBackoff = 0
	job := dto.Job{Title: "Engineer", URL: "https://example.com/1", BoardID: "11111111-1111-1111-1111-111111111111", ProviderPostingID: "posting-1"}
	if err := exporter.BulkExport(context.Background(), []dto.Job{job}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestAPIExporter_RequiresAcceptedOutcome(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"status":"pending"}]}`))
	}))
	defer srv.Close()
	exporter := NewAPIExporter(srv.URL, "tok")
	if err := exporter.BulkExport(context.Background(), []dto.Job{{Title: "x", URL: "https://x.com"}}); err == nil {
		t.Fatal("expected unknown outcome to fail")
	}
}

func TestAPIExporter_SplitsBatchAtPayloadLimit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request struct {
			Jobs []dto.Job `json:"jobs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if len(request.Jobs) != 1 {
			t.Errorf("batch %d has %d jobs, want 1", calls, len(request.Jobs))
		}
		_, _ = w.Write([]byte(`{"results":[{"status":"new","job_id":"job-1"}]}`))
	}))
	defer srv.Close()
	jobs := []dto.Job{
		{Title: "A", URL: "https://example.com/a", Description: strings.Repeat("x", 1100000)},
		{Title: "B", URL: "https://example.com/b", Description: strings.Repeat("x", 1100000)},
	}
	if err := NewAPIExporter(srv.URL, "tok").BulkExport(context.Background(), jobs); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
