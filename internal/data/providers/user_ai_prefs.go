package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type UserAIPrefsProvider interface {
	GetUserAIPrefs(ctx context.Context, userID string) (dto.UserAIPrefs, error)
	UpsertUserAIPrefs(ctx context.Context, userID, suitabilityModel string) (dto.UserAIPrefs, error)
}
