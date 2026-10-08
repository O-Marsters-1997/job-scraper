package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

const maxBodyBytes = 8 << 20

var hostCounter atomic.Uint32

func freshURL(path string) string {
	n := hostCounter.Add(1)
	return fmt.Sprintf("https://8.8.%d.%d%s", n>>8&255, n&255, path)
}

func response(status int, code string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"X-Brd-Err-Code": []string{code}}, Body: io.NopCloser(strings.NewReader("ok"))}
}

func newRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
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
	}), zone, nil)
	direct := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, ""), nil }), nil, nil)
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
	}), nil, nil)
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/", "http://10.0.0.1/", "http://169.254.169.254/"} {
		if _, err := tr.RoundTrip(newRequest(t, raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestFetchRejectsOversizedBody(t *testing.T) {
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxBodyBytes+1)))}, nil
	}), nil, nil)
	resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("oversized body should fail")
	}
}

func TestRateLimitIsNotRetriedAndDoesNotPauseZone(t *testing.T) {
	calls := 0
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(429, "client_10110"), nil
		}
		return response(200, ""), nil
	}), &proxy.ZoneGate{}, nil)
	var statuses []int
	for range 2 {
		resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/"))
		if err != nil {
			t.Fatal(err)
		}
		statuses = append(statuses, resp.StatusCode)
		_ = resp.Body.Close()
	}
	if diff := cmp.Diff([]int{429, 200}, statuses); diff != "" {
		t.Errorf("statuses (-want +got):\n%s", diff)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (no retry of the 429)", calls)
	}
}

func TestInFlightSuccessDoesNotResumeExhaustedZone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		inFlight := make(chan struct{})
		release := make(chan struct{})
		tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/slow" {
				close(inFlight)
				<-release
				return response(200, ""), nil
			}
			return response(502, "client_10100"), nil
		}), &proxy.ZoneGate{}, nil)
		slow := newRequest(t, freshURL("/slow"))
		exhaust := newRequest(t, freshURL("/exhaust"))
		next := newRequest(t, freshURL("/next"))
		go func() {
			if resp, err := tr.RoundTrip(slow); err == nil {
				_ = resp.Body.Close()
			}
		}()
		<-inFlight
		if _, err := tr.RoundTrip(exhaust); !proxy.IsZonePaused(err) {
			t.Fatalf("exhaustion = %v", err)
		}
		close(release)
		synctest.Wait()
		if _, err := tr.RoundTrip(next); !proxy.IsZonePaused(err) {
			t.Fatalf("late success resumed exhausted zone: %v", err)
		}
	})
}

func TestFetchLimitsConcurrentRequestsPerHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var entered atomic.Int32
		release := make(chan struct{})
		tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			entered.Add(1)
			<-release
			return response(200, ""), nil
		}), nil, nil)
		target := freshURL("/jobs")
		requests := []*http.Request{newRequest(t, target), newRequest(t, target), newRequest(t, target)}
		for _, req := range requests {
			go func() {
				if resp, err := tr.RoundTrip(req); err == nil {
					_ = resp.Body.Close()
				}
			}()
		}
		synctest.Wait()
		if got := entered.Load(); got != 2 {
			t.Fatalf("started %d same-host requests before a slot freed, want 2", got)
		}
		close(release)
		synctest.Wait()
		if got := entered.Load(); got != 3 {
			t.Errorf("started %d requests after a slot freed, want 3", got)
		}
	})
}

func TestFetchTransportCountsRequestsAndBytes(t *testing.T) {
	requests := proxy.FetchRequests.WithLabelValues("", "", "ok")
	bytes := proxy.FetchBytes.WithLabelValues("", "")
	beforeRequests, beforeBytes := testutil.ToFloat64(requests), testutil.ToFloat64(bytes)
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, ""), nil }), nil, nil)
	resp, err := tr.RoundTrip(newRequest(t, freshURL("/jobs")))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if got := testutil.ToFloat64(requests) - beforeRequests; got != 1 {
		t.Errorf("ok requests = %v, want 1", got)
	}
	if got := testutil.ToFloat64(bytes) - beforeBytes; got != 2 {
		t.Errorf("bytes = %v, want 2", got)
	}
}
