package identity

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func credentialKeyFromEnv() ([]byte, error) {
	enc := os.Getenv("AI_CREDENTIAL_ENC_KEY")
	if enc == "" {
		return nil, errors.New("AI_CREDENTIAL_ENC_KEY not set")
	}
	key, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("AI_CREDENTIAL_ENC_KEY: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("AI_CREDENTIAL_ENC_KEY: expected 32 bytes, got %d", len(key))
	}
	return key, nil
}

func (s *Service) UpdateCredential(ctx context.Context, userID string, in dto.UpsertCredentialInput) (struct{}, error) {
	if in.Provider == "" {
		return struct{}{}, apperr.Invalid("provider required")
	}
	if in.APIKey == nil {
		return struct{}{}, s.store.DeleteUserAICredential(ctx, userID, in.Provider)
	}
	enc, err := s.encrypt(strings.Trim(*in.APIKey, `"`))
	if err != nil {
		return struct{}{}, fmt.Errorf("identity.UpdateCredential: %w", err)
	}
	return struct{}{}, s.store.UpsertUserAICredential(ctx, userID, in.Provider, enc)
}

func (s *Service) GetCredential(ctx context.Context, userID, provider string) (string, error) {
	enc, err := s.store.GetUserAICredential(ctx, userID, provider)
	if err != nil {
		return "", err
	}
	plain, err := s.decrypt(enc)
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

func (s *Service) GetProfile(ctx context.Context, userID string) (dto.ProfileView, error) {
	p, err := s.store.GetProfile(ctx, userID)
	return dto.ProfileView(p), err
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, in dto.UpdateProfileInput) (struct{}, error) {
	_, err := s.store.UpdateEmail(ctx, userID, in.Email)
	return struct{}{}, err
}

func (s *Service) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.credKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *Service) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.credKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("ciphertext too short")
	}
	plain, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
