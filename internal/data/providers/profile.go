package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ProfileProvider is the single access point for user profile persistence.
type ProfileProvider interface {
	GetProfile(ctx context.Context, userID string) (dto.Profile, error)
	UpdateEmail(ctx context.Context, userID, email string) (dto.Profile, error)
}
