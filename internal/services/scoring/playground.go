package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const playgroundWindow = 30 * 24 * time.Hour

func (s *Service) PlaygroundQueue(ctx context.Context, userID string) (dto.PlaygroundQueue, error) {
	playgroundBands := []string{"great", "good", "fair", "poor"}
	candidates, err := s.store.ListPlaygroundCandidates(ctx, userID, jev.Model, time.Now().Add(-playgroundWindow))
	if err != nil {
		return dto.PlaygroundQueue{}, fmt.Errorf("scoring.PlaygroundQueue: %w", err)
	}
	pending := map[string][]string{}
	reviewed := map[string]int{}
	total := 0
	for _, c := range candidates {
		if c.Reviewed {
			reviewed[c.Band]++
			continue
		}
		pending[c.Band] = append(pending[c.Band], c.JobID)
		total++
	}
	queue := dto.PlaygroundQueue{JobIDs: make([]string, 0, total), Bands: make([]dto.PlaygroundBand, len(playgroundBands))}
	for i, band := range playgroundBands {
		queue.Bands[i] = dto.PlaygroundBand{Band: band, Reviewed: reviewed[band], Remaining: len(pending[band])}
	}
	for round := 0; len(queue.JobIDs) < total; round++ {
		for _, band := range playgroundBands {
			if round < len(pending[band]) {
				queue.JobIDs = append(queue.JobIDs, pending[band][round])
			}
		}
	}
	return queue, nil
}
