package scraper_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type knownURLs struct {
	known []string
	asked []string
}

func (k *knownURLs) NewURLs(_ context.Context, urls []string) ([]string, error) {
	k.asked = urls
	var out []string
	for _, u := range urls {
		if !slices.Contains(k.known, u) {
			out = append(out, u)
		}
	}
	return out, nil
}

type pageSourceStub struct {
	cards []dto.Job
	next  string
	got   *string
}

func (s pageSourceStub) Cfg() sources.Config { return sources.Config{Name: "wis"} }
func (s pageSourceStub) FetchPage(_ context.Context, cursor string) ([]dto.Job, string, error) {
	*s.got = cursor
	return s.cards, s.next, nil
}

type capturedCards struct{ cards []dto.Job }

func (c *capturedCards) CapturePage(_ context.Context, _ dto.SourceTarget, cards []dto.Job, _ dto.SearchConfig) error {
	c.cards = cards
	return nil
}

type noSearchConfig struct{}

func (noSearchConfig) SearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, data.ErrNotFound
}

func scrapePage(t *testing.T, known *knownURLs, cards []dto.Job, cursor string) (next, fetchedCursor string, captured []dto.Job) {
	t.Helper()
	capture := &capturedCards{}
	src := pageSourceStub{cards: cards, next: "2:5", got: &fetchedCursor}
	orch := scraper.New(known, noSearchConfig{}, func(dto.SourceTarget) (sources.Source, bool) { return src, true }, capture)
	next, err := orch.ScrapePage(t.Context(), dto.SourceTarget{ID: "target", UserID: "user", Source: "wis"}, cursor)
	if err != nil {
		t.Fatal(err)
	}
	return next, fetchedCursor, capture.cards
}

func TestScrapePage(t *testing.T) {
	const plain = "https://workinstartups.com/job/1"
	const aggregator = "https://www.linkedin.com/jobs/view/1?externalUrl=https://boards.greenhouse.io/acme/jobs/9"
	const rewritten = "https://boards.greenhouse.io/acme/jobs/9"

	t.Run("stops at the known job frontier", func(t *testing.T) {
		next, _, _ := scrapePage(t, &knownURLs{known: []string{plain}}, []dto.Job{{URL: plain}}, "")
		if next != "" {
			t.Errorf("next = %q, want empty", next)
		}
	})

	t.Run("passes the cursor in and the next cursor out", func(t *testing.T) {
		next, fetched, _ := scrapePage(t, &knownURLs{}, []dto.Job{{URL: plain}}, "1:5")
		if fetched != "1:5" || next != "2:5" {
			t.Errorf("fetched cursor = %q, next = %q, want 1:5, 2:5", fetched, next)
		}
	})

	t.Run("checks and captures rewritten URLs, skipping blanks", func(t *testing.T) {
		known := &knownURLs{}
		_, _, captured := scrapePage(t, known, []dto.Job{{URL: aggregator}, {URL: plain}, {URL: ""}}, "")
		if diff := cmp.Diff([]string{rewritten, plain}, known.asked); diff != "" {
			t.Errorf("URLs checked (-want +got):\n%s", diff)
		}
		var urls []string
		for _, card := range captured {
			urls = append(urls, card.URL)
		}
		if diff := cmp.Diff([]string{rewritten, plain, ""}, urls); diff != "" {
			t.Errorf("URLs captured (-want +got):\n%s", diff)
		}
	})
}
