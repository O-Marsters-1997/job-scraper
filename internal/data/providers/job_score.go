package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type JobScoreProvider interface {
	UpsertJobScoreRelevance(ctx context.Context, jobID, userID string, score int) error
	UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, score int, reasoning string, matched, missing []string) error
	GetJobScore(ctx context.Context, jobID, userID string) (dto.JobScore, error)
}
