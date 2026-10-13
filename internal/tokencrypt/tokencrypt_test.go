package tokencrypt_test

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"
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

var aad = tokencrypt.AAD{Table: "t", Owner: "owner"}

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
		ciphertext, err := c.Encrypt("hello world", aad)
		if err != nil {
			t.Fatalf("Encrypt error = %v", err)
		}
		return ciphertext
	}

	t.Run("round trips as v1", func(t *testing.T) {
		ciphertext := encrypt(t)
		if !strings.HasPrefix(ciphertext, "v1:") {
			t.Errorf("Encrypt() = %q, want v1: prefix", ciphertext)
		}
		got, err := c.Decrypt(ciphertext, aad)
		if err != nil {
			t.Fatalf("Decrypt error = %v", err)
		}
		if got != "hello world" {
			t.Errorf("Decrypt(Encrypt(%q)) = %q", "hello world", got)
		}
	})

	t.Run("rejects tampered ciphertext", func(t *testing.T) {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypt(t), "v1:"))
		if err != nil {
			t.Fatalf("DecodeString() err = %v", err)
		}
		raw[len(raw)-1] ^= 0xFF
		if _, err := c.Decrypt("v1:"+base64.StdEncoding.EncodeToString(raw), aad); err == nil {
			t.Error("Decrypt(tampered) err = nil, want error")
		}
	})

	t.Run("rejects wrong key", func(t *testing.T) {
		if _, err := newCipher(t, otherKey).Decrypt(encrypt(t), aad); err == nil {
			t.Error("Decrypt with another key err = nil, want error")
		}
	})

	t.Run("rejects another owner's row", func(t *testing.T) {
		other := tokencrypt.AAD{Table: "t", Owner: "someone-else"}
		if _, err := c.Decrypt(encrypt(t), other); err == nil {
			t.Error("Decrypt with another owner's AAD err = nil, want error")
		}
	})

	t.Run("rejects another table", func(t *testing.T) {
		other := tokencrypt.AAD{Table: "other", Owner: aad.Owner}
		if _, err := c.Decrypt(encrypt(t), other); err == nil {
			t.Error("Decrypt with another table's AAD err = nil, want error")
		}
	})

	t.Run("reads v0 ciphertext whatever the AAD", func(t *testing.T) {
		got, err := c.Decrypt(sealV0(t, validKey, "legacy"), aad)
		if err != nil || got != "legacy" {
			t.Errorf("Decrypt(v0) = %q, %v, want legacy", got, err)
		}
	})
}

func TestIsLegacy(t *testing.T) {
	c := newCipher(t, validKey)
	v1, err := c.Encrypt("x", aad)
	if err != nil {
		t.Fatal(err)
	}
	if tokencrypt.IsLegacy(v1) {
		t.Errorf("IsLegacy(%q) = true, want false", v1)
	}
	if v0 := sealV0(t, validKey, "x"); !tokencrypt.IsLegacy(v0) {
		t.Errorf("IsLegacy(%q) = false, want true", v0)
	}
}

func sealV0(t *testing.T, key, plaintext string) string {
	t.Helper()
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil))
}
