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

type boards struct {
	uk, abroad []dto.CompanyBoard
}

func (b boards) ListUntrackedDiscoveredBoards(_ context.Context, uk bool, _ time.Time) ([]dto.CompanyBoard, error) {
	if uk {
		return b.uk, nil
	}
	return b.abroad, nil
}

func TestHarvester(t *testing.T) {
	t.Parallel()
	lister := boards{
		uk:     []dto.CompanyBoard{{Source: "ashby", BoardToken: "acme"}},
		abroad: []dto.CompanyBoard{{Source: "greenhouse", BoardToken: "globex"}},
	}
	tests := []struct {
		name     string
		region   untracked.Region
		wantName string
		interval time.Duration
		want     discover.Harvest
	}{
		{"uk", untracked.UK, "untracked", 7 * 24 * time.Hour,
			discover.Harvest{Boards: []discover.Board{{Source: "ashby", Token: "acme"}}, Recheck: true}},
		{"abroad", untracked.Abroad, "untracked-abroad", 30 * 24 * time.Hour,
			discover.Harvest{Boards: []discover.Board{{Source: "greenhouse", Token: "globex"}}, Recheck: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := untracked.New(lister, tt.region)
			got, err := h.Harvest(t.Context())
			if err != nil {
				t.Fatalf("Harvest() err = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Harvest() mismatch (-want +got):\n%s", diff)
			}
			if h.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", h.Name(), tt.wantName)
			}
			if h.Interval() != tt.interval {
				t.Errorf("Interval() = %v, want %v", h.Interval(), tt.interval)
			}
		})
	}
}
