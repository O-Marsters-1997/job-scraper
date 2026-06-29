package credstore

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
)

// ErrNotFound is returned by Get when no credential exists for the given user+provider.
var ErrNotFound = errors.New("credential not found")

// CredentialStore persists encrypted AI provider keys per user.
type CredentialStore interface {
	Save(ctx context.Context, userID, provider, plainKey string) error
	Get(ctx context.Context, userID, provider string) (string, error)
	Delete(ctx context.Context, userID, provider string) error
	ListProviders(ctx context.Context, userID string) ([]string, error)
}

// RawStore is the persistence layer used by EnvCredentialStore.
// Method names mirror providers.UserAICredentialsProvider so that *db.DB
// satisfies this interface without an adapter.
type RawStore interface {
	UpsertUserAICredential(ctx context.Context, userID, provider, encKey string) error
	GetUserAICredential(ctx context.Context, userID, provider string) (string, error)
	DeleteUserAICredential(ctx context.Context, userID, provider string) error
	ListUserAICredentialProviders(ctx context.Context, userID string) ([]string, error)
}

// EnvCredentialStore is an AES-256-GCM CredentialStore that reads
// AI_CREDENTIAL_ENC_KEY (base64, must decode to 32 bytes) at construction.
//
// ponytail: key loaded once at New(); no per-op env reads.
type EnvCredentialStore struct {
	key []byte
	raw RawStore
}

// New constructs an EnvCredentialStore. Returns an error if
// AI_CREDENTIAL_ENC_KEY is missing or does not decode to 32 bytes.
func New(raw RawStore) (*EnvCredentialStore, error) {
	enc := os.Getenv("AI_CREDENTIAL_ENC_KEY")
	if enc == "" {
		return nil, errors.New("AI_CREDENTIAL_ENC_KEY not set")
	}
	k, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("AI_CREDENTIAL_ENC_KEY: %w", err)
	}
	if len(k) != 32 {
		return nil, fmt.Errorf("AI_CREDENTIAL_ENC_KEY: expected 32 bytes, got %d", len(k))
	}
	return &EnvCredentialStore{key: k, raw: raw}, nil
}

func (s *EnvCredentialStore) Save(ctx context.Context, userID, provider, plainKey string) error {
	enc, err := s.encrypt(plainKey)
	if err != nil {
		return fmt.Errorf("credstore.Save: %w", err)
	}
	return s.raw.UpsertUserAICredential(ctx, userID, provider, enc)
}

func (s *EnvCredentialStore) Get(ctx context.Context, userID, provider string) (string, error) {
	enc, err := s.raw.GetUserAICredential(ctx, userID, provider)
	if err != nil {
		return "", err
	}
	plain, err := s.decrypt(enc)
	if err != nil {
		return "", fmt.Errorf("credstore.Get: %w", err)
	}
	return plain, nil
}

func (s *EnvCredentialStore) Delete(ctx context.Context, userID, provider string) error {
	return s.raw.DeleteUserAICredential(ctx, userID, provider)
}

func (s *EnvCredentialStore) ListProviders(ctx context.Context, userID string) ([]string, error) {
	return s.raw.ListUserAICredentialProviders(ctx, userID)
}

func (s *EnvCredentialStore) encrypt(plaintext string) (string, error) {
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

func (s *EnvCredentialStore) decrypt(ciphertext string) (string, error) {
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
