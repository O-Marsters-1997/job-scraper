package tokencrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Cipher struct {
	aead cipher.AEAD
}

func New(base64Key string) (*Cipher, error) {
	if base64Key == "" {
		return nil, errors.New("key not set")
	}
	k, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, err
	}
	if len(k) != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", len(k))
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

func FromEnv(name string) (*Cipher, error) {
	c, err := New(os.Getenv(name))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return c, nil
}

const (
	TableAICredentials     = "user_ai_credentials"
	TableGoogleOAuthTokens = "google_oauth_tokens"
)

const v1Prefix = "v1:"

type AAD struct{ Table, Owner string }

func (a AAD) bytes() []byte { return []byte(a.Table + ":" + a.Owner) }

func IsLegacy(ciphertext string) bool { return !strings.HasPrefix(ciphertext, v1Prefix) }

func (c *Cipher) Encrypt(plaintext string, aad AAD) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), aad.bytes())
	return v1Prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) Decrypt(ciphertext string, aad AAD) (string, error) {
	encoded, v1 := strings.CutPrefix(ciphertext, v1Prefix)
	var ad []byte
	if v1 {
		ad = aad.bytes()
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	ns := c.aead.NonceSize()
	if len(data) < ns {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := c.aead.Open(nil, data[:ns], data[ns:], ad)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
