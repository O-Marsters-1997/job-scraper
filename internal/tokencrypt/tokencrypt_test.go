package tokencrypt

import (
	"encoding/base64"
	"testing"
)

var validKey = base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

func TestEncryptDecrypt(t *testing.T) {
	t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
	input := "hello world"

	ct, err := Encrypt(input)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != input {
		t.Errorf("Decrypt(Encrypt(%q)) = %q, want %q", input, got, input)
	}
}

func TestDecrypt_RejectsTamperedCiphertext(t *testing.T) {
	t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
	ct, err := Encrypt("hello world")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	raw, _ := base64.StdEncoding.DecodeString(ct)
	raw[len(raw)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := Decrypt(tampered); err == nil {
		t.Fatal("Decrypt(tampered) err = nil, want error")
	}
}

func TestDecrypt_RejectsWrongKey(t *testing.T) {
	t.Setenv("GOOGLE_TOKEN_ENC_KEY", validKey)
	ct, err := Encrypt("hello world")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	altKey := base64.StdEncoding.EncodeToString([]byte("99999999999999999999999999999999"))
	t.Setenv("GOOGLE_TOKEN_ENC_KEY", altKey)
	if _, err := Decrypt(ct); err == nil {
		t.Fatal("Decrypt(ct) err = nil, want error")
	}
}
