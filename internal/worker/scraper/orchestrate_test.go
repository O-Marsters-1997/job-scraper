package scraper_test

import (
	"context"
	"slices"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type knownURLs []string

func (k knownURLs) NewURLs(_ context.Context, urls []string) ([]string, error) {
	var out []string
	for _, u := range urls {
		if !slices.Contains(k, u) {
			out = append(out, u)
		}
	}
	return out, nil
}

type pageSourceStub struct {
	url string
}

func (s pageSourceStub) Cfg() sources.Config { return sources.Config{Name: "wis"} }
func (s pageSourceStub) FetchPage(context.Context, string) ([]dto.Job, string, error) {
	return []dto.Job{{URL: s.url}}, "2:5", nil
}

type emptyCandidateCapturer struct{}

func (emptyCandidateCapturer) CapturePage(context.Context, dto.SourceTarget, []dto.Job, dto.SearchConfig) error {
	return nil
}

type noSearchConfig struct{}

func (noSearchConfig) SearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, data.ErrNotFound
}

func TestScrapePageStopsAtKnownJobFrontier(t *testing.T) {
	ctx := context.Background()
	url := "https://workinstartups.com/job/1"
	build := func(dto.SourceTarget) (sources.Source, bool) { return pageSourceStub{url: url}, true }
	orch := scraper.New(knownURLs{url}, noSearchConfig{}, build, emptyCandidateCapturer{})
	next, err := orch.ScrapePage(ctx, dto.SourceTarget{ID: "target", UserID: "user", Source: "wis"}, "")
	if err != nil || next != "" {
		t.Fatalf("next=%q err=%v", next, err)
	}
}
