package scraper_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
)

const (
	okBody       = `{"results":[{"status":"new","job_id":"job-1"}]}`
	okSingleBody = `{"status":"new","job_id":"job-1"}`
)

var oneJob = []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}

func newExporter(t *testing.T, h http.HandlerFunc) *scraper.APIExporter {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return scraper.NewAPIExporter(srv.URL, "test-token").WithInitialBackoff(0)
}

func respondWith(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

type recordedRequest struct {
	Method, Path, ContentType, Auth string
	Jobs                            []dto.Job
}

func TestBulkExport(t *testing.T) {
	t.Run("posts the batch with auth", func(t *testing.T) {
		got := make(chan recordedRequest, 1)
		exporter := newExporter(t, func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Jobs []dto.Job `json:"jobs"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			got <- recordedRequest{r.Method, r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("Authorization"), body.Jobs}
			_, _ = w.Write([]byte(okBody))
		})
		if err := exporter.BulkExport(context.Background(), oneJob); err != nil {
			t.Fatal(err)
		}
		want := recordedRequest{http.MethodPost, "/ingest/batch", "application/json", "Bearer test-token", oneJob}
		if diff := cmp.Diff(want, <-got); diff != "" {
			t.Errorf("request (-want +got):\n%s", diff)
		}
	})

	t.Run("accepts any 2xx", func(t *testing.T) {
		exporter := newExporter(t, respondWith(http.StatusCreated, `{"results":[{"status":"unchanged"}]}`))
		if err := exporter.BulkExport(context.Background(), oneJob); err != nil {
			t.Errorf("BulkExport() = %v, want nil on 201", err)
		}
	})

	for _, tt := range []struct {
		name      string
		status    int
		body      string
		wantCalls int32
	}{
		{"4xx is not retried", http.StatusBadRequest, "", 1},
		{"5xx retries then fails", http.StatusServiceUnavailable, "", 3},
		{"unknown outcome", http.StatusOK, `{"results":[{"status":"pending"}]}`, 1},
		{"rejected job", http.StatusOK, `{"results":[{"status":"rejected","reason":"bad"}]}`, 1},
		{"result count mismatch", http.StatusOK, `{"results":[]}`, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			exporter := newExporter(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				respondWith(tt.status, tt.body)(w, r)
			})
			if err := exporter.BulkExport(context.Background(), oneJob); err == nil {
				t.Error("BulkExport() = nil, want error")
			}
			if got := calls.Load(); got != tt.wantCalls {
				t.Errorf("calls = %d, want %d", got, tt.wantCalls)
			}
		})
	}

	t.Run("retry resends the same body", func(t *testing.T) {
		var calls atomic.Int32
		bodies := make(chan string, 2)
		exporter := newExporter(t, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodies <- string(body)
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte(okBody))
		})
		if err := exporter.BulkExport(context.Background(), oneJob); err != nil {
			t.Fatal(err)
		}
		if first, second := <-bodies, <-bodies; first != second {
			t.Errorf("retry body changed: first=%s second=%s", first, second)
		}
	})

	t.Run("splits a batch at the payload limit", func(t *testing.T) {
		sizes := make(chan int, 4)
		exporter := newExporter(t, func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Jobs []dto.Job `json:"jobs"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			sizes <- len(body.Jobs)
			_, _ = w.Write([]byte(okBody))
		})
		big := strings.Repeat("x", 1100000)
		jobs := []dto.Job{
			{Title: "A", URL: "https://example.com/a", Description: big},
			{Title: "B", URL: "https://example.com/b", Description: big},
		}
		if err := exporter.BulkExport(context.Background(), jobs); err != nil {
			t.Fatal(err)
		}
		if first, second := <-sizes, <-sizes; first != 1 || second != 1 {
			t.Errorf("batch sizes = %d, %d, want 1, 1", first, second)
		}
	})
}

func TestExport(t *testing.T) {
	for _, tt := range []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"new", okSingleBody, false},
		{"rejected", `{"status":"rejected","reason":"bad"}`, true},
		{"unknown outcome", `{"status":"pending"}`, true},
		{"undecodable", `nope`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var path atomic.Value
			exporter := newExporter(t, func(w http.ResponseWriter, r *http.Request) {
				path.Store(r.URL.Path)
				_, _ = w.Write([]byte(tt.body))
			})
			err := exporter.Export(context.Background(), oneJob[0])
			if (err != nil) != tt.wantErr {
				t.Errorf("Export() error = %v, wantErr %t", err, tt.wantErr)
			}
			if got := path.Load(); got != "/ingest" {
				t.Errorf("path = %v, want /ingest", got)
			}
		})
	}
}
