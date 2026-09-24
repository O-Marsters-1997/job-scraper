package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func response(status int, code string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"X-Brd-Err-Code": []string{code}}, Body: io.NopCloser(strings.NewReader("ok"))}
}

func TestProtectedZonePausesOnlyOnExhaustion(t *testing.T) {
	now := time.Date(2026, 9, 23, 23, 50, 0, 0, time.UTC)
	zone := &zoneGate{now: func() time.Time { return now }}
	calls := 0
	protected := &fetchTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(502, "client_10100"), nil
		}
		return response(200, ""), nil
	}), zone: zone}
	direct := &fetchTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, ""), nil })}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://8.8.8.8/jobs", nil)
	if _, err := protected.RoundTrip(req); err == nil {
		t.Fatal("exhaustion should fail")
	}
	if _, err := protected.RoundTrip(req); err == nil {
		t.Fatal("protected request should stay paused")
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

func TestFetchRejectsUnsafeDestinationsAndLargeBody(t *testing.T) {
	tr := &fetchTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxBodyBytes+1)))}, nil
	})}
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/", "http://10.0.0.1/", "http://169.254.169.254/"} {
		req, _ := http.NewRequest(http.MethodGet, raw, nil)
		if _, err := tr.RoundTrip(req); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, "https://8.8.8.8/", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("oversized body should fail")
	}
}

func TestRateLimitDoesNotPauseZone(t *testing.T) {
	zone := &zoneGate{}
	calls := 0
	tr := &fetchTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(429, "client_10110"), nil
		}
		return response(200, ""), nil
	}), zone: zone}
	req, _ := http.NewRequest(http.MethodGet, "https://8.8.8.8/", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	resp, err = tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (one bounded retry)", calls)
	}
}

func TestInFlightSuccessDoesNotResumeExhaustedZone(t *testing.T) {
	zone := &zoneGate{}
	zone.result(true, false, false)
	zone.result(false, true, false)
	if _, err := zone.enter(); err == nil {
		t.Fatal("late success resumed exhausted zone")
	}
}

func TestFetchLimitsConcurrentRequestsPerHost(t *testing.T) {
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	tr := &fetchTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		entered <- struct{}{}
		<-release
		return response(200, ""), nil
	})}
	for range 3 {
		go func() {
			req, _ := http.NewRequest(http.MethodGet, "https://8.8.8.8/jobs", nil)
			resp, err := tr.RoundTrip(req)
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
