package cvtailor

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type CachedAsker interface {
	CachedAnswers(ctx context.Context, jobID string, questions []string) (map[string]dto.Answer, error)
}
