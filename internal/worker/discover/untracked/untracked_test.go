package untracked_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/untracked"
)

type boards []dto.CompanyBoard

func (b boards) ListUntrackedDiscoveredBoards(context.Context) ([]dto.CompanyBoard, error) {
	return b, nil
}

func TestHarvester(t *testing.T) {
	h := untracked.New(boards{{Source: "ashby", BoardToken: "acme"}, {Source: "greenhouse", BoardToken: "globex"}})
	got, err := h.Harvest(t.Context())
	if err != nil {
		t.Fatalf("Harvest() err = %v", err)
	}
	want := discover.Harvest{Boards: []discover.Board{{Source: "ashby", Token: "acme"}, {Source: "greenhouse", Token: "globex"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Harvest() mismatch (-want +got):\n%s", diff)
	}
	if got := h.Interval(); got != 7*24*time.Hour {
		t.Errorf("Interval() = %v, want 7 days", got)
	}
}
