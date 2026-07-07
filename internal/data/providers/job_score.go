package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type JobScoreProvider interface {
	UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, score int, reasoning string, matched, missing []string) error
	GetJobScore(ctx context.Context, jobID, userID string) (dto.JobScore, error)
}
