package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/api/services/jobs"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func seed(t *testing.T, store *providers.MockJobProvider, n int) {
	t.Helper()
	for i := range n {
		j := dto.Job{
			URL:       "https://example.com/" + string(rune('a'+i)),
			ScrapedAt: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC),
		}
		if _, err := store.Save(context.Background(), []dto.Job{j}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListRejectsBadPagination(t *testing.T) {
	svc := jobs.New(providers.NewMockJobProvider())
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
	store := providers.NewMockJobProvider()
	seed(t, store, 3)
	svc := jobs.New(store)

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
	svc := jobs.New(providers.NewMockJobProvider())
	_, err := svc.Get(context.Background(), "user-1", "missing")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
		t.Fatalf("status = %v, ok = %v, want 404", status, ok)
	}
}
