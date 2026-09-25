package scraper

import (
	"context"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources"
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

type emptyCandidateStore struct{}

func (emptyCandidateStore) SaveCards(context.Context, dto.SourceTarget, []dto.Job) ([]candidates.Candidate, error) {
	return nil, nil
}
func (emptyCandidateStore) ListForUser(context.Context, string, string, int) ([]candidates.Candidate, error) {
	return nil, nil
}
func (emptyCandidateStore) Assess(context.Context, string, string, time.Time, bool) (bool, error) {
	return false, nil
}
func (emptyCandidateStore) MarkDetailPending(context.Context, string) error { return nil }

func TestScrapePageStopsAtKnownJobFrontier(t *testing.T) {
	ctx := context.Background()
	url := "https://workinstartups.com/job/1"
	db := providers.NewMockJobProvider()
	if _, err := db.Save(ctx, []dto.Job{{URL: url}}); err != nil {
		t.Fatal(err)
	}
	orch := New(db, queue.NewMockQueue()).
		WithSourceBuilder(func(dto.SourceTarget) []sources.Source { return []sources.Source{pageSourceStub{url: url}} }).
		WithCandidates(emptyCandidateStore{})
	next, err := orch.ScrapePage(ctx, dto.SourceTarget{ID: "target", UserID: "user", Source: "wis"}, "")
	if err != nil || next != "" {
		t.Fatalf("next=%q err=%v", next, err)
	}
}
