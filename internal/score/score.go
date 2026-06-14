package score

import (
	"context"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// SuitabilityScorer scores a job's full description against the user's rubric.
type SuitabilityScorer interface {
	Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig) (int, TokenUsage, error)
}

// RelevanceScorer scores a job card against the user's search criteria.
type RelevanceScorer interface {
	Score(card dto.Job, cfg dto.SearchConfig) int
}

// TokenUsage records LLM token consumption (for SuitabilityScorer in a later issue).
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// HeuristicScorer is a pure, I/O-free keyword-matching scorer.
type HeuristicScorer struct{}

func NewHeuristicScorer() *HeuristicScorer { return &HeuristicScorer{} }

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

	if score > 100 {
		score = 100
	}
	return score
}
