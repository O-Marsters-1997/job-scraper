package recruitee_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func noGap() *sources.Gate { return sources.NewGate(0, time.Hour) }

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "offers_acme.json", recruitee.New("acme", noGap()))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "offers_acme.json", recruitee.New("acme", noGap())), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "offers_acme.json", got, want)
	}
}

func TestFetchPage_GoldenSalaryAndArrangement(t *testing.T) {
	sourcetest.RunGolden(t, "offers_tellent.json", recruitee.New("tellent", noGap()))
}

type refusingTransport struct{ requests int }

func (r *refusingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.requests++
	header := http.Header{"Retry-After": {"86400"}}
	return &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests", Header: header, Body: io.NopCloser(strings.NewReader(`{"error":"rate_limit"}`))}, nil
}

type replyTransport func(*http.Request) *http.Response

func (f replyTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req), nil }

func TestPollBoard_Retirement(t *testing.T) {
	tests := []struct {
		name  string
		reply func(*http.Request) *http.Response
		want  error
	}{
		{"a 404 is gone", func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("")), Request: req}
		}, sources.ErrGone},
		{"a redirect to another board is moved", func(req *http.Request) *http.Response {
			moved := req.Clone(req.Context())
			moved.URL.Host = "ballysintralotsa.recruitee.com"
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(`{"offers":[]}`)), Request: moved}
		}, sources.ErrMoved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := recruitee.New("intralot", noGap())
			src.Client().Transport = replyTransport(tt.reply)
			if _, err := src.PollBoard(t.Context()); !errors.Is(err, tt.want) {
				t.Errorf("PollBoard() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestPollBoard_RateLimitDefersLaterRequests(t *testing.T) {
	gate := noGap()
	tr := &refusingTransport{}
	first := recruitee.New("acme", gate)
	first.Client().Transport = tr
	_, err := first.PollBoard(t.Context())
	var status *sources.StatusError
	if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests {
		t.Fatalf("PollBoard() = %v, want a 429 StatusError", err)
	}
	other := recruitee.New("other", gate)
	other.Client().Transport = tr
	_, err = other.PollBoard(t.Context())
	if !errors.Is(err, sources.ErrDeferred) {
		t.Errorf("PollBoard() after the 429 = %v, want ErrDeferred", err)
	}
	if tr.requests != 1 {
		t.Errorf("sent %d requests, want only the one that was refused", tr.requests)
	}
}
