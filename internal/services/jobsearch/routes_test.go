package jobsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type fakeIngestStore struct {
	byURL  map[string]dto.Job
	nextID int
}

func newFakeIngestStore() *fakeIngestStore {
	return &fakeIngestStore{byURL: make(map[string]dto.Job)}
}

func (f *fakeIngestStore) SaveCanonical(_ context.Context, job dto.Job) (dto.Job, string, error) {
	if existing, ok := f.byURL[job.URL]; ok {
		return existing, "unchanged", nil
	}
	f.nextID++
	job.ID = fmtID(f.nextID)
	f.byURL[job.URL] = job
	return job, "new", nil
}

func (f *fakeIngestStore) UpsertCompany(_ context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	return dto.Company{Slug: c.Slug, Name: c.Name}, nil
}

func fmtID(n int) string {
	digits := "0123456789"
	s := ""
	for n > 0 {
		s = string(digits[n%10]) + s
		n /= 10
	}
	if s == "" {
		s = "0"
	}
	return "job-" + s
}

func newIngestRouter(store *fakeIngestStore) http.Handler {
	m := &Module{ingest: newIngester(store, store)}
	r := chi.NewRouter()
	m.PublicRoutes(r)
	return r
}

func TestIngestHandlerRejectsMalformedBody(t *testing.T) {
	handler := newIngestRouter(newFakeIngestStore())
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestIngestHandlerRepeatedDeliveryKeepsOneCanonicalJob(t *testing.T) {
	store := newFakeIngestStore()
	handler := newIngestRouter(store)
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
		var res IngestResult
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
	if len(store.byURL) != 1 {
		t.Fatalf("stored %d jobs, want 1", len(store.byURL))
	}
}

func TestIngestBatchHandlerReturnsOrderedOutcomes(t *testing.T) {
	handler := newIngestRouter(newFakeIngestStore())
	body := `{"jobs":[{"title":"Engineer","url":"https://example.com/1"},{"title":"","url":"https://example.com/2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var view ingestBatchView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Results) != 2 || view.Results[0].Status != "new" || view.Results[1].Status != "rejected" {
		t.Fatalf("results = %+v", view.Results)
	}
}
