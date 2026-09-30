package proxy_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

const maxBodyBytes = 8 << 20

func response(status int, code string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"X-Brd-Err-Code": []string{code}}, Body: io.NopCloser(strings.NewReader("ok"))}
}

func newRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func TestProtectedZonePausesOnlyOnExhaustion(t *testing.T) {
	now := time.Date(2026, 9, 23, 23, 50, 0, 0, time.UTC)
	zone := &proxy.ZoneGate{Now: func() time.Time { return now }}
	calls := 0
	protected := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(502, "client_10100"), nil
		}
		return response(200, ""), nil
	}), zone)
	direct := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, ""), nil }), nil)
	req := newRequest(t, "https://8.8.8.8/jobs")
	if _, err := protected.RoundTrip(req); !proxy.IsZonePaused(err) {
		t.Fatalf("exhaustion should pause source: %v", err)
	}
	if _, err := protected.RoundTrip(req); !proxy.IsZonePaused(err) {
		t.Fatalf("protected request should stay paused: %v", err)
	}
	if calls != 1 {
		t.Fatalf("protected calls = %d, want 1", calls)
	}
	resp, err := direct.RoundTrip(req)
	if err != nil {
		t.Fatalf("direct source paused: %v", err)
	}
	_ = resp.Body.Close()
	now = now.Add(20 * time.Minute)
	resp, err = protected.RoundTrip(req)
	if err != nil {
		t.Fatalf("daily probe failed: %v", err)
	}
	_ = resp.Body.Close()
	if calls != 2 {
		t.Fatalf("protected calls = %d, want 2", calls)
	}
}

func TestFetchRejectsUnsafeDestinations(t *testing.T) {
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, ""), nil
	}), nil)
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/", "http://10.0.0.1/", "http://169.254.169.254/"} {
		if _, err := tr.RoundTrip(newRequest(t, raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestFetchRejectsOversizedBody(t *testing.T) {
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxBodyBytes+1)))}, nil
	}), nil)
	resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("oversized body should fail")
	}
}

func TestRateLimitDoesNotPauseZone(t *testing.T) {
	calls := 0
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(429, "client_10110"), nil
		}
		return response(200, ""), nil
	}), &proxy.ZoneGate{})
	for range 2 {
		resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/"))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (one bounded retry)", calls)
	}
}

func TestInFlightSuccessDoesNotResumeExhaustedZone(t *testing.T) {
	inFlight := make(chan struct{})
	release := make(chan struct{})
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/slow" {
			close(inFlight)
			<-release
			return response(200, ""), nil
		}
		return response(502, "client_10100"), nil
	}), &proxy.ZoneGate{})
	slowDone := make(chan struct{})
	go func() {
		defer close(slowDone)
		if resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/slow")); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-inFlight
	if _, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/exhaust")); !proxy.IsZonePaused(err) {
		t.Fatalf("exhaustion = %v", err)
	}
	close(release)
	<-slowDone
	if _, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/next")); !proxy.IsZonePaused(err) {
		t.Fatalf("late success resumed exhausted zone: %v", err)
	}
}

func TestFetchLimitsConcurrentRequestsPerHost(t *testing.T) {
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		entered <- struct{}{}
		<-release
		return response(200, ""), nil
	}), nil)
	for range 3 {
		go func() {
			resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/jobs"))
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
	}
	<-entered
	<-entered
	select {
	case <-entered:
		t.Fatal("third same-host request started before a slot freed")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("third request did not start after a slot freed")
	}
}
