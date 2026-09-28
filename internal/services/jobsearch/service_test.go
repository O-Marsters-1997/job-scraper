package jobsearch_test

import (
	"context"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func seedJobs(n int) *jobsearchtest.FakeStore {
	st := jobsearchtest.NewFakeStore()
	for i := range n {
		if _, _, err := st.SaveCanonical(context.Background(), dto.Job{
			Title: "Role", URL: "https://example.com/" + string(rune('a'+i)),
			ScrapedAt: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC),
		}); err != nil {
			panic(err)
		}
	}
	return st
}

func TestListRejectsBadPagination(t *testing.T) {
	svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, nil)
	for _, q := range []dto.JobsQuery{
		{Limit: "9999"},
		{Cursor: "not-base64"},
		{Availability: "unknown"},
	} {
		if _, err := svc.List(context.Background(), "user-1", q); err == nil {
			t.Errorf("q = %+v: want error", q)
		} else if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
			t.Errorf("q = %+v: status = %v, ok = %v, want 400", q, status, ok)
		}
	}
}

func TestListPaginates(t *testing.T) {
	svc := jobsearch.NewService(seedJobs(3), nil, nil)

	page, err := svc.List(context.Background(), "user-1", dto.JobsQuery{Limit: "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("page = %+v, want 2 items and a cursor", page)
	}

	next, err := svc.List(context.Background(), "user-1", dto.JobsQuery{Limit: "2", Cursor: page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Items) != 1 || next.NextCursor != "" {
		t.Fatalf("next page = %+v, want 1 item and no cursor", next)
	}
}

func TestGetMapsNotFound(t *testing.T) {
	svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, nil)
	_, err := svc.Get(context.Background(), "user-1", "missing")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
		t.Fatalf("status = %v, ok = %v, want 404", status, ok)
	}
}
