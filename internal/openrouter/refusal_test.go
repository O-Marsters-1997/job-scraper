package openrouter_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

func TestRefusal(t *testing.T) {
	t.Parallel()
	status := func(code int, body string) error {
		return &openrouter.StatusError{Code: code, Body: body}
	}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"401", status(http.StatusUnauthorized, ""), "OpenRouter rejected your API key; check it in Settings, AI"},
		{"402 with reason", status(http.StatusPaymentRequired, `{"error":{"message":"fewer max_tokens"}}`),
			"OpenRouter says your account is out of credit for this request; top up or lower the key's limit in Settings, AI: fewer max_tokens"},
		{"403 unparsable body", status(http.StatusForbidden, "<html>"),
			"OpenRouter refused the request (key limit exceeded or content moderated); check your key in Settings, AI"},
	}
	seen := map[string]bool{}
	for _, tt := range tests {
		got := openrouter.Refusal(tt.err)
		if !apperr.IsKind(got, apperr.KindUnprocessable) || got.Error() != tt.want {
			t.Errorf("%s: got %v, want %q", tt.name, got, tt.want)
		}
		if seen[got.Error()] {
			t.Errorf("%s: message not distinct", tt.name)
		}
		seen[got.Error()] = true
	}
	if openrouter.Refusal(status(http.StatusInternalServerError, "")) != nil || openrouter.Refusal(errors.New("x")) != nil {
		t.Error("non-refusal errors must return nil")
	}
}
