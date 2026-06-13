package tokencrypt

import (
	"encoding/base64"
	"testing"
)

var validKey = base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		input   string
		wantErr bool
	}{
		{
			name: "round-trip",
			setup: func(t *testing.T) {
				t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
			},
			input: "hello world",
		},
		{
			name: "tampered ciphertext",
			setup: func(t *testing.T) {
				t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
			},
			input:   "hello world",
			wantErr: true,
		},
		{
			name: "wrong key",
			setup: func(t *testing.T) {
				t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
			},
			input:   "hello world",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			ct, err := Encrypt(tc.input)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			switch tc.name {
			case "round-trip":
				got, err := Decrypt(ct)
				if err != nil {
					t.Fatalf("Decrypt: %v", err)
				}
				if got != tc.input {
					t.Errorf("got %q, want %q", got, tc.input)
				}

			case "tampered ciphertext":
				raw, _ := base64.StdEncoding.DecodeString(ct)
				raw[len(raw)-1] ^= 0xFF
				tampered := base64.StdEncoding.EncodeToString(raw)
				_, err := Decrypt(tampered)
				if err == nil {
					t.Fatal("expected error for tampered ciphertext, got nil")
				}

			case "wrong key":
				altKey := base64.StdEncoding.EncodeToString([]byte("99999999999999999999999999999999"))
				t.Setenv("GOOGLE_TOKEN_ENC_KEY", altKey)
				_, err := Decrypt(ct)
				if err == nil {
					t.Fatal("expected error for wrong key, got nil")
				}
			}
		})
	}
}
