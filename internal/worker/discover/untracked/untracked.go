// Package untracked re-harvests discovered Boards no User tracks, so a Board
// that has since gained a matching role is tracked.
package untracked

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const interval = 7 * 24 * time.Hour

type BoardLister interface {
	ListUntrackedDiscoveredBoards(ctx context.Context) ([]dto.CompanyBoard, error)
}

type Harvester struct {
	boards BoardLister
}

func New(boards BoardLister) *Harvester {
	return &Harvester{boards: boards}
}

func (h *Harvester) Name() string            { return "untracked" }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	rows, err := h.boards.ListUntrackedDiscoveredBoards(ctx)
	if err != nil {
		return discover.Harvest{}, fmt.Errorf("list untracked boards: %w", err)
	}
	out := discover.Harvest{Boards: make([]discover.Board, len(rows))}
	for i, b := range rows {
		out.Boards[i] = discover.Board{Source: b.Source, Token: b.BoardToken}
	}
	return out, nil
}
