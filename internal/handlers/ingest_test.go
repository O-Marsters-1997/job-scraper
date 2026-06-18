package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

type mockScorer struct {
	calls int
}

func (m *mockScorer) ScoreAndSave(_ context.Context, _ dto.Job) int {
	m.calls++
	return 42
}

type mockNotifier struct {
	calls int
}

func (m *mockNotifier) NotifyNewJob(_ context.Context, _ dto.Job, _ int) {
	m.calls++
}

func buildHandler(db ingest.Saver, scorer ingest.Scorer, notifier ingest.Notifier) http.Handler {
	ing := ingest.New(db, scorer, notifier)
	h := NewIngestHandler(ing)
	return auth.ServiceTokenMiddleware(http.HandlerFunc(h.Ingest))
}

func TestIngestHandler(t *testing.T) {
	const goodToken = "good-token"
	const validBody = `{"title":"Engineer","url":"https://example.com/1"}`

	tests := []struct {
		name       string
		authHeader string
		body       string
		wantStatus int
	}{
		{
			name:       "no token",
			authHeader: "",
			body:       validBody,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong token",
			authHeader: "Bearer bad-token",
			body:       validBody,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token and job",
			authHeader: "Bearer " + goodToken,
			body:       validBody,
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed body",
			authHeader: "Bearer " + goodToken,
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("INGEST_SERVICE_TOKEN", goodToken)
			db := providers.NewMockJobProvider()
			handler := buildHandler(db, nil, nil)

			req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("want status %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestIngestHandler_SaveCalled(t *testing.T) {
	t.Setenv("INGEST_SERVICE_TOKEN", "tok")
	db := providers.NewMockJobProvider()
	scorer := &mockScorer{}
	notifier := &mockNotifier{}
	handler := buildHandler(db, scorer, notifier)

	body := `{"title":"Engineer","url":"https://example.com/job1"}`
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"ingested":1}` {
		t.Errorf("want body {\"ingested\":1}, got %q", got)
	}
	jobs := db.Jobs()
	if len(jobs) != 1 {
		t.Fatalf("want 1 saved job, got %d", len(jobs))
	}
	if scorer.calls != 1 {
		t.Errorf("want scorer called once, got %d", scorer.calls)
	}
	if notifier.calls != 1 {
		t.Errorf("want notifier called once, got %d", notifier.calls)
	}
}

func TestIngestHandler_DuplicateURL(t *testing.T) {
	t.Setenv("INGEST_SERVICE_TOKEN", "tok")
	db := providers.NewMockJobProvider()
	scorer := &mockScorer{}
	notifier := &mockNotifier{}
	handler := buildHandler(db, scorer, notifier)

	body := `{"title":"Engineer","url":"https://example.com/job2"}`

	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer tok")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}
	}

	jobs := db.Jobs()
	if len(jobs) != 1 {
		t.Errorf("want 1 saved job after duplicate, got %d", len(jobs))
	}
}
