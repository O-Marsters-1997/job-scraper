package jobsearch_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func newIngestRouter(t *testing.T, st *jobsearchtest.FakeStore) http.Handler {
	t.Helper()
	m := jobsearch.Build(jobsearchtest.NewDeps(st))
	r := chi.NewRouter()
	m.PublicRoutes(r)
	return r
}

func TestRoutesRequireAuth(t *testing.T) {
	r := chi.NewRouter()
	jobsearch.Build(jobsearchtest.NewDeps(jobsearchtest.NewFakeStore())).Routes(r)
	handlerstest.RequiresAuth(t, r,
		"GET /jobs", "GET /jobs/all", "GET /jobs/{id}",
		"GET /sources", "GET /sources/resolve",
		"GET /source-targets", "POST /source-targets", "PATCH /source-targets/{id}",
		"POST /source-targets/{id}/scrape", "DELETE /source-targets/{id}",
		"GET /companies", "GET /companies/tracked", "POST /companies",
		"PUT /companies/{id}/tracking", "DELETE /companies/{id}/tracking",
		"GET /companies/{id}/boards", "POST /companies/{id}/boards",
	)
}

func TestIngestHandlerRejectsMalformedBody(t *testing.T) {
	handlerstest.RejectsMalformedBody(t, newIngestRouter(t, jobsearchtest.NewFakeStore()), "POST /ingest")
}

func TestIngestHandlerRepeatedDeliveryKeepsOneCanonicalJob(t *testing.T) {
	handler := newIngestRouter(t, jobsearchtest.NewFakeStore())
	body := `{"title":"Engineer","url":"https://example.com/job2"}`

	var firstID string
	for i := range 2 {
		req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body.String())
		}
		var res jobsearch.IngestResult
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstID = res.JobID
			if res.Status != "new" {
				t.Fatalf("first delivery status = %q, want new", res.Status)
			}
		} else if res.Status != "unchanged" || res.JobID != firstID {
			t.Fatalf("second delivery = %+v, want unchanged with job_id %q", res, firstID)
		}
	}
}

func TestIngestHandlerRejectsInvalidJobs(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{"non-http url scheme", `{"Title":"Engineer","URL":"file:///tmp/job"}`},
		{"malformed board id", `{"Title":"Engineer","URL":"https://example.com/jobs/bad-board","BoardID":"not-a-uuid"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler := newIngestRouter(t, jobsearchtest.NewFakeStore())
			req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", w.Code, w.Body.String())
			}
			var res jobsearch.IngestResult
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatal(err)
			}
			if res.Status != "rejected" {
				t.Fatalf("result = %+v, want rejected", res)
			}
		})
	}
}

func TestIngestHandlerRejectsConflictingBoardIdentity(t *testing.T) {
	handler := newIngestRouter(t, jobsearchtest.NewFakeStore())
	first := `{"Title":"Engineer","URL":"https://example.com/conflict","BoardID":"11111111-1111-1111-1111-111111111111","ProviderPostingID":"posting-1"}`
	conflict := `{"Title":"Engineer","URL":"https://example.com/conflict","BoardID":"22222222-2222-2222-2222-222222222222","ProviderPostingID":"posting-2"}`

	for _, body := range []string{first, conflict} {
		req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body.String())
		}
	}
}

func TestIngestBatchHandlerReturnsOrderedOutcomes(t *testing.T) {
	handler := newIngestRouter(t, jobsearchtest.NewFakeStore())
	body := `{"jobs":[{"title":"Engineer","url":"https://example.com/1"},{"title":"","url":"https://example.com/2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var view struct {
		Results []jobsearch.IngestResult `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Results) != 2 || view.Results[0].Status != "new" || view.Results[1].Status != "rejected" {
		t.Fatalf("results = %+v", view.Results)
	}
}
