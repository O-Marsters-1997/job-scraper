package score

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func newAnthropicError(t *testing.T, status int, header http.Header) error {
	t.Helper()
	return &anthropic.Error{
		StatusCode: status,
		Request:    httptest.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", nil),
		Response:   &http.Response{StatusCode: status, Header: header},
	}
}

func TestClassifyAnthropicError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		header         http.Header
		wantKind       FailureKind
		wantRetryAfter time.Duration
	}{
		{"unauthorized", http.StatusUnauthorized, nil, FailureTerminal, 0},
		{"no credit", http.StatusPaymentRequired, nil, FailureTerminal, 0},
		{"rate limited with retry-after", http.StatusTooManyRequests, http.Header{"Retry-After": []string{"120"}}, FailureRateLimited, 120 * time.Second},
		{"rate limited without retry-after", http.StatusTooManyRequests, nil, FailureRetryable, 0},
		{"server error", http.StatusInternalServerError, nil, FailureRetryable, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyAnthropicError(newAnthropicError(t, tt.status, tt.header))

			var scorerErr *ScorerError
			if errors.As(err, &scorerErr) {
				if scorerErr.Kind != tt.wantKind {
					t.Fatalf("kind = %v, want %v", scorerErr.Kind, tt.wantKind)
				}
				if scorerErr.RetryAfter != tt.wantRetryAfter {
					t.Fatalf("retryAfter = %v, want %v", scorerErr.RetryAfter, tt.wantRetryAfter)
				}
				return
			}
			if tt.wantKind != FailureRetryable {
				t.Fatalf("expected classified error of kind %v, got plain error %v", tt.wantKind, err)
			}
		})
	}
}

func TestClassifyAnthropicError_NonAPIError(t *testing.T) {
	err := classifyAnthropicError(errors.New("connection reset"))

	var scorerErr *ScorerError
	if errors.As(err, &scorerErr) {
		t.Fatalf("non-API error should not be classified, got %+v", scorerErr)
	}
	if err == nil {
		t.Fatal("expected wrapped error")
	}
}
