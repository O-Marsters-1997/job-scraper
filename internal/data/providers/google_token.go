package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ErrGoogleTokenNotFound is returned when no OAuth token exists for the user.
var ErrGoogleTokenNotFound = errors.New("google token not found")

// GoogleTokenProvider persists Google OAuth tokens.
type GoogleTokenProvider interface {
	GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error)
	UpsertGoogleToken(ctx context.Context, input dto.UpsertGoogleTokenInput) error
	DeleteGoogleToken(ctx context.Context, userID string) error
}
