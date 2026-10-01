package sources_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func TestProtectedSourceDoesNotFetchDirectlyWithoutCredentials(t *testing.T) {
	t.Setenv("BRIGHTDATA_PROXY_URL", "")
	src := sources.NewBase(sources.Config{Name: "protected", Route: sources.RouteUnlocker})
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
