package identitytest

import (
	"encoding/base64"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

func NewCipher(t *testing.T) *tokencrypt.Cipher {
	t.Helper()
	c, err := tokencrypt.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
