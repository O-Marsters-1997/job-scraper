package companies

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sources"
)

type pageSourceFunc func(context.Context, string) (sources.Page, error)

func (f pageSourceFunc) FetchPage(ctx context.Context, cursor string) (sources.Page, error) {
	return f(ctx, cursor)
}

func TestVerifyPagesEnumeratesSynchronously(t *testing.T) {
	var cursors []string
	err := verifyPages(t.Context(), pageSourceFunc(func(_ context.Context, cursor string) (sources.Page, error) {
		cursors = append(cursors, cursor)
		if cursor == "" {
			return sources.Page{NextCursor: "second"}, nil
		}
		return sources.Page{}, nil
	}))
	if err != nil || len(cursors) != 2 || cursors[0] != "" || cursors[1] != "second" {
		t.Fatalf("cursors = %v, err = %v", cursors, err)
	}
}

func TestVerifyPagesStopsOnError(t *testing.T) {
	want := errors.New("fetch failed")
	err := verifyPages(t.Context(), pageSourceFunc(func(context.Context, string) (sources.Page, error) {
		return sources.Page{}, want
	}))
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyPagesStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	requests := 0
	err := verifyPages(ctx, pageSourceFunc(func(context.Context, string) (sources.Page, error) {
		requests++
		cancel()
		return sources.Page{NextCursor: "again"}, nil
	}))
	if !errors.Is(err, context.Canceled) || requests != 1 {
		t.Fatalf("requests = %d, err = %v", requests, err)
	}
}

func TestVerifierRejectsUnsupportedBoards(t *testing.T) {
	for _, pair := range [][2]string{{"missing", "acme"}, {"remoteok", "acme"}, {"greenhouse", ""}, {"greenhouse", "a/b"}} {
		if err := (ATSBoardVerifier{}).Verify(t.Context(), pair[0], pair[1]); err == nil {
			t.Errorf("accepted %q %q", pair[0], pair[1])
		}
	}
}
