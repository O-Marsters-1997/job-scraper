package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) UpdateCredential(ctx context.Context, userID string, in dto.UpsertCredentialInput) (struct{}, error) {
	provider, err := dto.ParseProvider(string(in.Provider))
	if err != nil {
		return struct{}{}, err
	}
	if in.APIKey == nil {
		err := s.store.DeleteUserAICredential(ctx, userID, provider)
		s.usage.drop(userID)
		return struct{}{}, err
	}
	enc, err := s.cipher.Encrypt(strings.Trim(*in.APIKey, `"`))
	if err != nil {
		return struct{}{}, fmt.Errorf("identity.UpdateCredential: %w", err)
	}
	err = s.store.UpsertUserAICredential(ctx, userID, provider, enc)
	s.usage.drop(userID)
	return struct{}{}, err
}

func (s *Service) GetCredential(ctx context.Context, userID string, provider dto.Provider) (string, error) {
	enc, err := s.store.GetUserAICredential(ctx, userID, provider)
	if err != nil {
		return "", err
	}
	plain, err := s.cipher.Decrypt(enc)
	if err != nil {
		return "", fmt.Errorf("identity.GetCredential: %w", err)
	}
	return plain, nil
}

func (s *Service) GetAIPrefs(ctx context.Context, userID string) (dto.AIPrefsView, error) {
	configured, err := s.store.ListUserAICredentialProviders(ctx, userID)
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

func (s *Service) UpdateProfile(ctx context.Context, userID string, in dto.UpdateProfileInput) (struct{}, error) {
	_, err := s.store.UpdateEmail(ctx, userID, in.Email)
	return struct{}{}, err
}
