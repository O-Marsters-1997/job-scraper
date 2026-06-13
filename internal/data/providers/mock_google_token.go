package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type MockGoogleTokenProvider struct {
	token *dto.GoogleToken
}

func (m *MockGoogleTokenProvider) GetGoogleToken(_ context.Context, _ string) (dto.GoogleToken, error) {
	if m.token == nil {
		return dto.GoogleToken{}, ErrGoogleTokenNotFound
	}
	return *m.token, nil
}

func (m *MockGoogleTokenProvider) UpsertGoogleToken(_ context.Context, input dto.UpsertGoogleTokenInput) error {
	m.token = &dto.GoogleToken{
		AccessTokenEnc:  input.AccessTokenEnc,
		RefreshTokenEnc: input.RefreshTokenEnc,
		TokenType:       input.TokenType,
		Expiry:          input.Expiry,
		Scope:           input.Scope,
	}
	return nil
}

func (m *MockGoogleTokenProvider) DeleteGoogleToken(_ context.Context, _ string) error {
	m.token = nil
	return nil
}

func (m *MockGoogleTokenProvider) Seed(tok dto.GoogleToken) {
	m.token = &tok
}
