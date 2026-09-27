package sources_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/sources"
)

func TestList(t *testing.T) {
	got, err := sources.New().List(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("want at least one registered source")
	}
}

func TestResolve(t *testing.T) {
	svc := sources.New()

	_, err := svc.Resolve(context.Background(), "user-1", dto.ResolveBoardQuery{})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("empty url: status = %v, ok = %v", status, ok)
	}

	_, err = svc.Resolve(context.Background(), "user-1", dto.ResolveBoardQuery{URL: "https://example.com/careers"})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnprocessable.Status() {
		t.Fatalf("unresolvable url: status = %v, ok = %v", status, ok)
	}

	got, err := svc.Resolve(context.Background(), "user-1", dto.ResolveBoardQuery{URL: "https://boards.greenhouse.io/acme"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "greenhouse" || got.Value != "acme" {
		t.Fatalf("got = %+v", got)
	}
}
