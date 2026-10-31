package cvtailor

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/outreachhook"
)

type CachedAsker interface {
	CachedAnswers(ctx context.Context, jobID string, questions []string) (map[string]dto.Answer, error)
}

type HookWriter interface {
	Write(ctx context.Context, apiKey string, in outreachhook.Input) (outreachhook.Result, error)
}

var _ HookWriter = (*outreachhook.Writer)(nil)
