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

type Region int

const (
	UK Region = iota
	Abroad
)

const (
	ukWindow       = 30 * 24 * time.Hour
	ukInterval     = 7 * 24 * time.Hour
	abroadInterval = 30 * 24 * time.Hour
)

type BoardLister interface {
	ListUntrackedDiscoveredBoards(ctx context.Context, uk bool, ukSince time.Time) ([]dto.CompanyBoard, error)
}

type Harvester struct {
	boards BoardLister
	region Region
}

func New(boards BoardLister, region Region) *Harvester {
	return &Harvester{boards: boards, region: region}
}

func (h *Harvester) Name() string {
	if h.region == Abroad {
		return "untracked-abroad"
	}
	return "untracked"
}

func (h *Harvester) Interval() time.Duration {
	if h.region == Abroad {
		return abroadInterval
	}
	return ukInterval
}

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	rows, err := h.boards.ListUntrackedDiscoveredBoards(ctx, h.region == UK, time.Now().Add(-ukWindow))
	if err != nil {
		return discover.Harvest{}, fmt.Errorf("list untracked boards: %w", err)
	}
	out := discover.Harvest{Boards: make([]discover.Board, len(rows)), Recheck: true}
	for i, b := range rows {
		out.Boards[i] = discover.Board{Source: b.Source, Token: b.BoardToken}
	}
	return out, nil
}
