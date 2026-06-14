package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrGoogleTokenNotFound = errors.New("google token not found")

type GoogleTokenProvider interface {
	GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error)
	UpsertGoogleToken(ctx context.Context, input dto.UpsertGoogleTokenInput) error
	DeleteGoogleToken(ctx context.Context, userID string) error
}
