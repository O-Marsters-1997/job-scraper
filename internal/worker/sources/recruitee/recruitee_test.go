package recruitee_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "offers_acme.json", recruitee.New("acme"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "offers_acme.json", recruitee.New("acme")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "offers_acme.json", got, want)
	}
}

func TestFetchPage_GoldenSalaryAndArrangement(t *testing.T) {
	sourcetest.RunGolden(t, "offers_tellent.json", recruitee.New("tellent"))
}

type refusingTransport struct{ requests int }

func (r *refusingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.requests++
	header := http.Header{"Retry-After": {"86400"}}
	return &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests", Header: header, Body: io.NopCloser(strings.NewReader(`{"error":"rate_limit"}`))}, nil
}

func TestPollBoard_RateLimitDefersLaterRequests(t *testing.T) {
	recruitee.ResetLimiter()
	tr := &refusingTransport{}
	first := recruitee.New("acme")
	first.Client().Transport = tr
	_, err := first.PollBoard(t.Context())
	var status *sources.StatusError
	if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests {
		t.Fatalf("PollBoard() = %v, want a 429 StatusError", err)
	}
	other := recruitee.New("other")
	other.Client().Transport = tr
	_, err = other.PollBoard(t.Context())
	if !errors.Is(err, sources.ErrDeferred) {
		t.Errorf("PollBoard() after the 429 = %v, want ErrDeferred", err)
	}
	if tr.requests != 1 {
		t.Errorf("sent %d requests, want only the one that was refused", tr.requests)
	}
}
