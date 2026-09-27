package jobsearch_test

import (
	"context"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

type fakeJobStore struct {
	jobs []dto.Job
}

func seedJobs(n int) *fakeJobStore {
	f := &fakeJobStore{}
	for i := range n {
		f.jobs = append(f.jobs, dto.Job{
			ID:        string(rune('a' + i)),
			URL:       "https://example.com/" + string(rune('a'+i)),
			ScrapedAt: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC),
		})
	}
	return f
}

func (f *fakeJobStore) Page(_ context.Context, _ string, options dto.JobPageOptions) (dto.JobPage, error) {
	items := make([]dto.Job, len(f.jobs))
	copy(items, f.jobs)
	if options.CursorID != "" {
		for i, j := range items {
			if j.ID == options.CursorID {
				items = items[i+1:]
				break
			}
		}
	}
	limit := int(options.Limit)
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return dto.JobPage{Items: items}, nil
}

func (f *fakeJobStore) GetJob(_ context.Context, id, _ string) (dto.Job, error) {
	for _, j := range f.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return dto.Job{}, store.ErrNotFound
}

func TestListRejectsBadPagination(t *testing.T) {
	svc := jobsearch.NewService(&fakeJobStore{})
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
	svc := jobsearch.NewService(seedJobs(3))

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
	svc := jobsearch.NewService(&fakeJobStore{})
	_, err := svc.Get(context.Background(), "user-1", "missing")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
		t.Fatalf("status = %v, ok = %v, want 404", status, ok)
	}
}
