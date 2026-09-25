package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/auth"
	"github.com/ollymarsters/job-scraper/internal/api/ingest"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

func buildHandler(db ingest.CanonicalSaver) http.Handler {
	ing := ingest.New(db, nil)
	return auth.ServiceTokenMiddleware(Ingest(ing))
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
			handler := buildHandler(db)

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
	handler := buildHandler(db)

	body := `{"title":"Engineer","url":"https://example.com/job1"}`
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"status":"new","job_id":"mock-1"}`+"\n" {
		t.Errorf("unexpected response %q", got)
	}
	jobs := db.Jobs()
	if len(jobs) != 1 {
		t.Fatalf("want 1 saved job, got %d", len(jobs))
	}
}

func TestIngestHandler_DuplicateURL(t *testing.T) {
	t.Setenv("INGEST_SERVICE_TOKEN", "tok")
	db := providers.NewMockJobProvider()
	handler := buildHandler(db)

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

func TestIngestBatch_ReturnsOrderedOutcomes(t *testing.T) {
	t.Setenv("INGEST_SERVICE_TOKEN", "tok")
	db := providers.NewMockJobProvider()
	ing := ingest.New(db, nil)
	handler := auth.ServiceTokenMiddleware(IngestBatch(ing))

	body := `{"jobs":[{"title":"Engineer","url":"https://example.com/1"},{"title":"","url":"https://example.com/2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != `{"results":[{"status":"new","job_id":"mock-1"},{"status":"rejected","reason":"title and url are required"}]}`+"\n" {
		t.Fatalf("response = %q", got)
	}
}

func TestIngestBatch_RejectsInvalidIdentityWithoutLosingValidJob(t *testing.T) {
	t.Setenv("INGEST_SERVICE_TOKEN", "tok")
	db := providers.NewMockJobProvider()
	handler := auth.ServiceTokenMiddleware(IngestBatch(ingest.New(db, nil)))
	body := `{"jobs":[{"title":"Bad","url":"file:///etc/passwd"},{"title":"Bad","url":"https://example.com/1","BoardID":"invalid"},{"title":"Good","url":"https://example.com/2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Results []ingest.Result `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 3 || response.Results[0].Status != "rejected" || response.Results[1].Status != "rejected" || response.Results[2].Status != "new" {
		t.Fatalf("results = %+v", response.Results)
	}
	if len(db.Jobs()) != 1 {
		t.Fatalf("saved %d jobs, want 1", len(db.Jobs()))
	}
}

func TestIngestBatch_ReturnsDistinctCanonicalIDs(t *testing.T) {
	t.Setenv("INGEST_SERVICE_TOKEN", "tok")
	handler := auth.ServiceTokenMiddleware(IngestBatch(ingest.New(providers.NewMockJobProvider(), nil)))
	request := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewBufferString(`{"jobs":[{"title":"A","url":"https://example.com/a"},{"title":"B","url":"https://example.com/b"}]}`))
	request.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)
	var response struct {
		Results []ingest.Result `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 2 || response.Results[0].JobID == response.Results[1].JobID {
		t.Fatalf("canonical IDs = %+v", response.Results)
	}
}
