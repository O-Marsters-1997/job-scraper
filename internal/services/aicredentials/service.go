// Package aicredentials is the identity context's per-user AI provider key
// storage, encrypted at rest with AES-256-GCM.
package aicredentials

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

// Store is the identity store's user_ai_credentials CRUD, dealing in
// already-encrypted keys.
type Store interface {
	UpsertUserAICredential(ctx context.Context, userID, provider, encKey string) error
	GetUserAICredential(ctx context.Context, userID, provider string) (string, error)
	DeleteUserAICredential(ctx context.Context, userID, provider string) error
	ListUserAICredentialProviders(ctx context.Context, userID string) ([]string, error)
}

// Service encrypts and decrypts provider API keys with a key read from
// AI_CREDENTIAL_ENC_KEY (base64, must decode to 32 bytes) at construction.
type Service struct {
	key   []byte
	store Store
}

// New reads AI_CREDENTIAL_ENC_KEY and returns an error if it is missing or
// does not decode to 32 bytes.
func New(store Store) (*Service, error) {
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
	return &Service{key: key, store: store}, nil
}

// Update saves in.APIKey for in.Provider, or clears it when APIKey is nil.
func (s *Service) Update(ctx context.Context, userID string, in dto.UpsertCredentialInput) (struct{}, error) {
	if in.Provider == "" {
		return struct{}{}, apperr.Invalid("provider required")
	}
	if in.APIKey == nil {
		return struct{}{}, s.store.DeleteUserAICredential(ctx, userID, in.Provider)
	}
	enc, err := s.encrypt(strings.Trim(*in.APIKey, `"`))
	if err != nil {
		return struct{}{}, fmt.Errorf("aicredentials.Update: %w", err)
	}
	return struct{}{}, s.store.UpsertUserAICredential(ctx, userID, in.Provider, enc)
}

// Get returns userID's decrypted key for provider.
func (s *Service) Get(ctx context.Context, userID, provider string) (string, error) {
	enc, err := s.store.GetUserAICredential(ctx, userID, provider)
	if err != nil {
		return "", err
	}
	plain, err := s.decrypt(enc)
	if err != nil {
		return "", fmt.Errorf("aicredentials.Get: %w", err)
	}
	return plain, nil
}

// ListProviders returns the providers userID has a configured key for.
func (s *Service) ListProviders(ctx context.Context, userID string) ([]string, error) {
	return s.store.ListUserAICredentialProviders(ctx, userID)
}

func (s *Service) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
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
	block, err := aes.NewCipher(s.key)
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
