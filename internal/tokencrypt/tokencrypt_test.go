package tokencrypt_test

import (
	"encoding/base64"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

func newCipher(t *testing.T, key string) *tokencrypt.Cipher {
	t.Helper()
	c, err := tokencrypt.New(base64.StdEncoding.EncodeToString([]byte(key)))
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	return c
}

const (
	validKey = "12345678901234567890123456789012"
	otherKey = "99999999999999999999999999999999"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"missing", ""},
		{"not base64", "!!!"},
		{"wrong length", base64.StdEncoding.EncodeToString(make([]byte, 16))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tokencrypt.New(tt.key); err == nil {
				t.Errorf("New(%q) err = nil, want error", tt.key)
			}
		})
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("TOKENCRYPT_TEST_KEY", base64.StdEncoding.EncodeToString([]byte(validKey)))
	if _, err := tokencrypt.FromEnv("TOKENCRYPT_TEST_KEY"); err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if _, err := tokencrypt.FromEnv("TOKENCRYPT_TEST_UNSET"); err == nil {
		t.Error("FromEnv(unset) err = nil, want error")
	}
}

func TestDecrypt(t *testing.T) {
	c := newCipher(t, validKey)
	encrypt := func(t *testing.T) string {
		t.Helper()
		ciphertext, err := c.Encrypt("hello world")
		if err != nil {
			t.Fatalf("Encrypt error = %v", err)
		}
		return ciphertext
	}

	t.Run("round trips", func(t *testing.T) {
		got, err := c.Decrypt(encrypt(t))
		if err != nil {
			t.Fatalf("Decrypt error = %v", err)
		}
		if got != "hello world" {
			t.Errorf("Decrypt(Encrypt(%q)) = %q", "hello world", got)
		}
	})

	t.Run("rejects tampered ciphertext", func(t *testing.T) {
		raw, err := base64.StdEncoding.DecodeString(encrypt(t))
		if err != nil {
			t.Fatalf("DecodeString() err = %v", err)
		}
		raw[len(raw)-1] ^= 0xFF
		if _, err := c.Decrypt(base64.StdEncoding.EncodeToString(raw)); err == nil {
			t.Error("Decrypt(tampered) err = nil, want error")
		}
	})

	t.Run("rejects wrong key", func(t *testing.T) {
		if _, err := newCipher(t, otherKey).Decrypt(encrypt(t)); err == nil {
			t.Error("Decrypt with another key err = nil, want error")
		}
	})
}
