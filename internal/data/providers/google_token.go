package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrGoogleTokenNotFound = errors.New("google token not found")

// ErrGoogleTokenUnusable is returned when a token row exists but cannot be
// decrypted (e.g. bad or rotated GOOGLE_TOKEN_ENC_KEY). The handler treats
// this as "not connected" rather than an internal error.
var ErrGoogleTokenUnusable = errors.New("google token unusable")

type GoogleTokenProvider interface {
	GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error)
	UpsertGoogleToken(ctx context.Context, input dto.UpsertGoogleTokenInput) error
	DeleteGoogleToken(ctx context.Context, userID string) error
}
