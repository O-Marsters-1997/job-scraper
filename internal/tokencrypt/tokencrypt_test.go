package tokencrypt_test

import (
	"encoding/base64"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

const keyEnv = "GOOGLE_TOKEN_ENC_KEY"

var (
	validKey = base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	otherKey = base64.StdEncoding.EncodeToString([]byte("99999999999999999999999999999999"))
)

func encrypted(t *testing.T, plaintext string) string {
	t.Helper()
	t.Setenv(keyEnv, validKey)
	ciphertext, err := tokencrypt.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt(%q) error = %v", plaintext, err)
	}
	return ciphertext
}

func TestDecrypt(t *testing.T) {
	t.Run("round trips", func(t *testing.T) {
		got, err := tokencrypt.Decrypt(encrypted(t, "hello world"))
		if err != nil {
			t.Fatalf("Decrypt error = %v", err)
		}
		if got != "hello world" {
			t.Errorf("Decrypt(Encrypt(%q)) = %q", "hello world", got)
		}
	})

	t.Run("rejects tampered ciphertext", func(t *testing.T) {
		raw, err := base64.StdEncoding.DecodeString(encrypted(t, "hello world"))
		if err != nil {
			t.Fatal(err)
		}
		raw[len(raw)-1] ^= 0xFF
		if _, err := tokencrypt.Decrypt(base64.StdEncoding.EncodeToString(raw)); err == nil {
			t.Fatal("Decrypt(tampered) error = nil, want error")
		}
	})

	t.Run("rejects wrong key", func(t *testing.T) {
		ciphertext := encrypted(t, "hello world")
		t.Setenv(keyEnv, otherKey)
		if _, err := tokencrypt.Decrypt(ciphertext); err == nil {
			t.Fatal("Decrypt with another key error = nil, want error")
		}
	})
}
