package scraper

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type pageSourceStub struct {
	url string
}

func (s pageSourceStub) Cfg() sources.Config { return sources.Config{Name: "wis"} }
func (s pageSourceStub) Iterate(context.Context, func(context.Context, []dto.Job) (bool, error)) error {
	return nil
}
func (s pageSourceStub) FetchPage(context.Context, string) ([]dto.Job, string, error) {
	return []dto.Job{{URL: s.url}}, "2:5", nil
}

type emptyCandidateCapturer struct{}

func (emptyCandidateCapturer) CapturePage(context.Context, dto.SourceTarget, []dto.Job, dto.SearchConfig) error {
	return nil
}

func TestScrapePageStopsAtKnownJobFrontier(t *testing.T) {
	ctx := context.Background()
	url := "https://workinstartups.com/job/1"
	db := providers.NewMockJobProvider()
	if _, err := db.Save(ctx, []dto.Job{{URL: url}}); err != nil {
		t.Fatal(err)
	}
	orch := New(db).
		WithSourceBuilder(func(dto.SourceTarget) []sources.Source { return []sources.Source{pageSourceStub{url: url}} }).
		WithCandidates(emptyCandidateCapturer{})
	next, err := orch.ScrapePage(ctx, dto.SourceTarget{ID: "target", UserID: "user", Source: "wis"}, "")
	if err != nil || next != "" {
		t.Fatalf("next=%q err=%v", next, err)
	}
}
