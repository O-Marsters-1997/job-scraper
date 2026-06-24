package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type JobScoreProvider interface {
	UpsertJobScoreRelevance(ctx context.Context, jobID, userID string, sc int) error
	UpsertJobScoreSuitability(ctx context.Context, s score.SuitabilityScore) error
	GetJobScore(ctx context.Context, jobID, userID string) (dto.JobScore, error)
}
