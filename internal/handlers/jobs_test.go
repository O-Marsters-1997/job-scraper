package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type mockDB struct {
	jobs []dto.Job
	err  error
}

func (m *mockDB) Save(_ context.Context, _ []dto.Job) error                  { return nil }
func (m *mockDB) NewURLs(_ context.Context, urls []string) ([]string, error) { return urls, nil }
func (m *mockDB) List(_ context.Context) ([]dto.Job, error)                  { return m.jobs, m.err }

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
			db:         &mockDB{jobs: []dto.Job{fixedJob}},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "empty list returns empty array",
			db:         &mockDB{jobs: []dto.Job{}},
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
			h := New(tt.db)
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
			var got []dto.Job
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("want %d jobs, got %d", tt.wantLen, len(got))
			}
		})
	}
}
