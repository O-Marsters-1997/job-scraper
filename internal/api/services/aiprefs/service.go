package aiprefs

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type CredentialLister interface {
	ListProviders(ctx context.Context, userID string) ([]string, error)
}

type Service struct {
	creds CredentialLister
}

func New(creds CredentialLister) *Service {
	return &Service{creds: creds}
}

func (s *Service) Get(ctx context.Context, userID string) (dto.AIPrefsView, error) {
	configured, err := s.creds.ListProviders(ctx, userID)
	if err != nil {
		return dto.AIPrefsView{}, err
	}
	if configured == nil {
		configured = []string{}
	}
	return dto.AIPrefsView{
		ConfiguredProviders: configured,
		ScoringEnabled:      len(configured) > 0,
	}, nil
}
