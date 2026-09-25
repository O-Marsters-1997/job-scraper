// Package aicredentials holds the domain rule for storing or clearing a
// user's AI provider API key.
package aicredentials

import (
	"context"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Store interface {
	Save(ctx context.Context, userID, provider, plainKey string) error
	Delete(ctx context.Context, userID, provider string) error
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

// Update saves in.APIKey for in.Provider, or clears it when APIKey is nil.
func (s *Service) Update(ctx context.Context, userID string, in dto.UpsertCredentialInput) (struct{}, error) {
	if in.Provider == "" {
		return struct{}{}, apperr.Invalid("provider required")
	}
	if in.APIKey == nil {
		return struct{}{}, s.store.Delete(ctx, userID, in.Provider)
	}
	return struct{}{}, s.store.Save(ctx, userID, in.Provider, strings.Trim(*in.APIKey, `"`))
}
