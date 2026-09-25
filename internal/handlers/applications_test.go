package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestGetApplicationsForJobsNoIDsReturnsEmptyObject(t *testing.T) {
	h := NewApplicationHandler(providers.NewMockApplicationProvider())
	req := withSession(httptest.NewRequest(http.MethodGet, "/applications/for-jobs", nil), "user-1")
	w := httptest.NewRecorder()
	h.GetApplicationsForJobs(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "{}\n" {
		t.Fatalf("body = %q, want {}", got)
	}
}

func TestListApplicationsFiltersByStatusWhenGiven(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	if _, err := store.CreateApplication(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"}); err != nil {
		t.Fatal(err)
	}
	h := NewApplicationHandler(store)
	req := withSession(httptest.NewRequest(http.MethodGet, "/applications?status_id=missing", nil), "user-1")
	w := httptest.NewRecorder()
	h.ListApplications(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want []", got)
	}
}
