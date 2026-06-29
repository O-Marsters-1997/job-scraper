package score

import (
	"context"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SuitabilityResult struct {
	Score     int
	Matched   []string
	Missing   []string
	Rationale string
	Usage     TokenUsage
}

// SuitabilityScorer scores job descriptions against the user's rubric.
// modelID selects which Claude model to use; if empty, the implementation uses its default.
type SuitabilityScorer interface {
	Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error)
	// ScoreBatch scores multiple jobs in fewer API calls. Results are in the same order as jobs.
	ScoreBatch(ctx context.Context, jobs []dto.Job, cfg dto.SearchConfig, modelID string) ([]SuitabilityResult, error)
}

type RelevanceScorer interface {
	Score(card dto.Job, cfg dto.SearchConfig) int
}

type TokenUsage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	CostUSD             float64
}

// HeuristicScorer is a pure, I/O-free keyword-matching scorer.
type HeuristicScorer struct{}

func (h *HeuristicScorer) Score(card dto.Job, cfg dto.SearchConfig) int {
	score := 0
	titleLower := strings.ToLower(card.Title)
	locationLower := strings.ToLower(card.Location)

	if cfg.Role != "" && strings.Contains(titleLower, strings.ToLower(cfg.Role)) {
		score += 50
	}

	if len(cfg.Keywords) > 0 {
		matched := 0
		for _, kw := range cfg.Keywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				matched++
			}
		}
		score += 30 * matched / len(cfg.Keywords)
	}

	if cfg.Location != "" && strings.Contains(locationLower, strings.ToLower(cfg.Location)) {
		score += 20
	}

	return min(score, 100)
}
