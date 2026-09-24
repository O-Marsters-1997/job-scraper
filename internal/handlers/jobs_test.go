package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type mockDB struct {
	jobs    []dto.Job
	err     error
	page    providers.JobPage
	options providers.JobPageOptions
}

func (m *mockDB) Save(_ context.Context, jobs []dto.Job) ([]dto.Job, error)  { return jobs, nil }
func (m *mockDB) NewURLs(_ context.Context, urls []string) ([]string, error) { return urls, nil }
func (m *mockDB) List(_ context.Context, _ string) ([]dto.Job, error)        { return m.jobs, m.err }
func (m *mockDB) Page(_ context.Context, _ string, options providers.JobPageOptions) (providers.JobPage, error) {
	m.options = options
	return m.page, m.err
}
func (m *mockDB) GetJob(_ context.Context, _ string, _ string) (dto.Job, error) {
	if len(m.jobs) == 0 {
		return dto.Job{}, providers.ErrNotFound
	}
	return m.jobs[0], nil
}
func (m *mockDB) ListSince(_ context.Context, _ time.Time) ([]dto.Job, error) { return m.jobs, m.err }

func TestListJobs(t *testing.T) {
	fixedJob := dto.Job{
		ID:          "abc-123",
		Title:       "Product Engineer",
		Location:    "London",
		URL:         "https://example.com/job/1",
		CompanySlug: "acme",
		Source:      "wis",
		UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		ScrapedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name       string
		db         *mockDB
		wantStatus int
		wantLen    int
	}{
		{
			name:       "returns jobs as JSON",
			db:         &mockDB{page: providers.JobPage{Items: []dto.Job{fixedJob}}},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "empty list returns empty array",
			db:         &mockDB{page: providers.JobPage{Items: []dto.Job{}}},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "db error returns 500",
			db:         &mockDB{err: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewJobHandler(tt.db)
			req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
			w := httptest.NewRecorder()

			h.ListJobs(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("want application/json, got %s", ct)
			}
			var got providers.JobPage
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(got.Items) != tt.wantLen {
				t.Fatalf("want %d jobs, got %d", tt.wantLen, len(got.Items))
			}
		})
	}
}

func TestListJobsPagination(t *testing.T) {
	first := dto.Job{ID: "00000000-0000-0000-0000-000000000002", ScrapedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}
	second := dto.Job{ID: "00000000-0000-0000-0000-000000000001", ScrapedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	db := &mockDB{page: providers.JobPage{Items: []dto.Job{first, second}}}
	h := NewJobHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/jobs?limit=1&availability=open", nil)
	w := httptest.NewRecorder()
	h.ListJobs(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	var got providers.JobPage
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != first.ID || got.NextCursor == "" {
		t.Fatalf("page: %+v", got)
	}
	if db.options.Limit != 2 || db.options.Availability != "open" {
		t.Fatalf("options: %+v", db.options)
	}
	req = httptest.NewRequest(http.MethodGet, "/jobs?limit=1&cursor="+got.NextCursor, nil)
	w = httptest.NewRecorder()
	h.ListJobs(w, req)
	if w.Code != http.StatusOK || !db.options.CursorTime.Equal(first.ScrapedAt) || db.options.CursorID != first.ID {
		t.Fatalf("cursor: %d %+v", w.Code, db.options)
	}
}

func TestListJobsRejectsBadPagination(t *testing.T) {
	h := NewJobHandler(&mockDB{})
	for _, url := range []string{"/jobs?limit=9999", "/jobs?cursor=bad", "/jobs?availability=unknown"} {
		w := httptest.NewRecorder()
		h.ListJobs(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", url, w.Code)
		}
	}
}

func TestGetJob(t *testing.T) {
	db := &mockDB{jobs: []dto.Job{{ID: "job-1", Description: "full text"}}}
	r := chi.NewRouter()
	r.Get("/jobs/{id}", NewJobHandler(db).GetJob)
	req := httptest.NewRequest(http.MethodGet, "/jobs/job-1", nil)
	req = req.WithContext(auth.WithSession(req.Context(), dto.Session{UserID: "user-1"}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !json.Valid(w.Body.Bytes()) {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}
