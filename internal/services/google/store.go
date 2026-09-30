// Package google is the identity context's Google Link sibling feature: the
// OAuth2 client and the Google Docs/Drive surface cvtemplates consumes
// (ADR 0011).
package google

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrTokenNotFound = apperr.Unauthorized("google account not connected")

// ErrTokenUnusable means a token row exists but cannot be decrypted (e.g.
// bad or rotated GOOGLE_TOKEN_ENC_KEY).
var ErrTokenUnusable = errors.New("google token unusable")

// Store is the identity store's google_oauth_tokens CRUD; identity's store
// implements it and returns ErrTokenNotFound when userID has no row
// (ADR 0011).
type Store interface {
	GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error)
	UpsertGoogleToken(ctx context.Context, input dto.UpsertGoogleTokenInput) error
	DeleteGoogleToken(ctx context.Context, userID string) error
}
