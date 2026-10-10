package sources_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func TestProtectedSourceDoesNotFetchDirectlyWithoutCredentials(t *testing.T) {
	t.Setenv("DECODO_PROXY_URL", "")
	src := sources.NewBase(sources.Config{Name: "protected", Route: sources.RouteResidential})
	if _, err := src.Get(t.Context(), "https://8.8.8.8/jobs"); err == nil {
		t.Fatal("protected fetch should fail without credentials")
	}
}

func TestFetcherRejectsUnsafeRedirect(t *testing.T) {
	src := sources.NewBase(sources.Config{Name: "direct"})
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/private", nil)
	if err := src.Client().CheckRedirect(req, nil); err == nil {
		t.Fatal("private redirect should be rejected")
	}
}

type statusTransport int

func (s statusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: int(s), Status: http.StatusText(int(s)), Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
}

func TestGetMapsGoneStatuses(t *testing.T) {
	tests := []struct {
		status   int
		wantGone bool
	}{
		{http.StatusNotFound, true},
		{http.StatusGone, true},
		{http.StatusForbidden, false},
		{http.StatusInternalServerError, false},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			src := sources.NewBase(sources.Config{Name: "direct"})
			src.Client().Transport = statusTransport(tt.status)
			_, err := src.Get(t.Context(), "https://example.com/job")
			if err == nil {
				t.Fatal("Get() = nil, want error")
			}
			if got := errors.Is(err, sources.ErrGone); got != tt.wantGone {
				t.Fatalf("errors.Is(err, ErrGone) = %v, want %v", got, tt.wantGone)
			}
		})
	}
}

type retryAfterTransport string

func (h retryAfterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := http.Header{}
	if h != "" {
		header.Set("Retry-After", string(h))
	}
	return &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429 Too Many Requests",
		Header:     header,
		Body:       http.NoBody,
		Request:    req,
	}, nil
}

func TestGetCarriesRetryAfterOnStatusError(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		atLeast time.Duration
		atMost  time.Duration
	}{
		{"delta seconds", "120", 120 * time.Second, 120 * time.Second},
		{"http date", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat), 59 * time.Minute, time.Hour},
		{"missing", "", 0, 0},
		{"overflowing seconds", "9000000000000", 24 * time.Hour, 24 * time.Hour},
		{"garbage", "soon", 0, 0},
		{"past date", time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := sources.NewBase(sources.Config{Name: "direct"})
			src.Client().Transport = retryAfterTransport(tt.header)
			_, err := src.Get(t.Context(), "https://example.com/jobs")
			var statusErr *sources.StatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("Get() error = %v, want *StatusError", err)
			}
			if got := statusErr.RetryAfter; got < tt.atLeast || got > tt.atMost {
				t.Errorf("RetryAfter = %v, want in [%v, %v]", got, tt.atLeast, tt.atMost)
			}
		})
	}
}
