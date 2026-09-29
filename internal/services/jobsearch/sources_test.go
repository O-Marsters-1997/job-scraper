package jobsearch_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
)

func TestListSources(t *testing.T) {
	got, err := jobsearch.NewService(nil, nil).ListSources(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("want at least one registered source")
	}
}

func TestResolveBoard(t *testing.T) {
	svc := jobsearch.NewService(nil, nil)

	_, err := svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("empty url: status = %v, ok = %v", status, ok)
	}

	_, err = svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{URL: "https://example.com/careers"})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnprocessable.Status() {
		t.Fatalf("unresolvable url: status = %v, ok = %v", status, ok)
	}

	got, err := svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{URL: "https://boards.greenhouse.io/acme"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "greenhouse" || got.Value != "acme" {
		t.Fatalf("got = %+v", got)
	}
}
