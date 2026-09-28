package tokencrypt

import (
	"encoding/base64"
	"testing"
)

var validKey = base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, ciphertext string)
	}{
		{
			name: "round trip",
			check: func(t *testing.T, ciphertext string) {
				t.Helper()
				got, err := Decrypt(ciphertext)
				if err != nil {
					t.Fatalf("Decrypt: %v", err)
				}
				if got != "hello world" {
					t.Errorf("Decrypt(Encrypt(%q)) = %q, want %q", "hello world", got, "hello world")
				}
			},
		},
		{
			name: "rejects tampered ciphertext",
			check: func(t *testing.T, ciphertext string) {
				t.Helper()
				raw, _ := base64.StdEncoding.DecodeString(ciphertext)
				raw[len(raw)-1] ^= 0xFF
				tampered := base64.StdEncoding.EncodeToString(raw)
				if _, err := Decrypt(tampered); err == nil {
					t.Fatal("Decrypt(tampered) err = nil, want error")
				}
			},
		},
		{
			name: "rejects wrong key",
			check: func(t *testing.T, ciphertext string) {
				t.Helper()
				altKey := base64.StdEncoding.EncodeToString([]byte("99999999999999999999999999999999"))
				t.Setenv("GOOGLE_TOKEN_ENC_KEY", altKey)
				if _, err := Decrypt(ciphertext); err == nil {
					t.Fatal("Decrypt(ct) err = nil, want error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
			ct, err := Encrypt("hello world")
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			tt.check(t, ct)
		})
	}
}
