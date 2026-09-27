// Package profile is the identity context's user profile read/update.
package profile

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Store interface {
	GetProfile(ctx context.Context, userID string) (dto.Profile, error)
	UpdateEmail(ctx context.Context, userID, email string) (dto.Profile, error)
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(ctx context.Context, userID string) (dto.ProfileView, error) {
	p, err := s.store.GetProfile(ctx, userID)
	return dto.ProfileView(p), err
}

func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateProfileInput) (struct{}, error) {
	_, err := s.store.UpdateEmail(ctx, userID, in.Email)
	return struct{}{}, err
}
