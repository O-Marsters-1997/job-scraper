package proxy_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

const maxBodyBytes = proxy.DefaultMaxBody

var hostCounter atomic.Uint32

func freshURL(path string) string {
	n := hostCounter.Add(1)
	return fmt.Sprintf("https://8.8.%d.%d%s", n>>8&255, n&255, path)
}

func response(status int) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok"))}
}

func newRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func TestFetchRejectsUnsafeDestinations(t *testing.T) {
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200), nil
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

func TestNewFetchTransportMax(t *testing.T) {
	const limit = 2 * maxBodyBytes
	read := func(t *testing.T, size int) error {
		t.Helper()
		tr := proxy.NewFetchTransportMax(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", size)))}, nil
		}), nil, limit)
		resp, err := tr.RoundTrip(newRequest(t, "https://8.8.8.8/"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		_, err = io.ReadAll(resp.Body)
		return err
	}

	t.Run("body past the default but within the raised cap is read", func(t *testing.T) {
		if err := read(t, maxBodyBytes+1); err != nil {
			t.Errorf("read body = %v, want nil", err)
		}
	})
	t.Run("body past the raised cap is a MaxBytesError", func(t *testing.T) {
		var tooLarge *http.MaxBytesError
		if err := read(t, limit+1); !errors.As(err, &tooLarge) {
			t.Errorf("read body = %v, want *http.MaxBytesError", err)
		}
	})
}

func TestRateLimitIsNotRetried(t *testing.T) {
	calls := 0
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(429), nil
		}
		return response(200), nil
	}), nil)
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

func TestFetchLimitsConcurrentRequestsPerHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var entered atomic.Int32
		release := make(chan struct{})
		tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			entered.Add(1)
			<-release
			return response(200), nil
		}), nil)
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
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) { return response(200), nil }), nil)
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

func TestFetchCachesOnlyGetAndHead(t *testing.T) {
	for _, tc := range []struct {
		method    string
		wantCalls int
	}{
		{http.MethodGet, 1},
		{http.MethodHead, 1},
		{http.MethodPost, 2},
	} {
		t.Run(tc.method, func(t *testing.T) {
			calls := 0
			cache := &memCache{entries: map[string]dto.CachedResponse{}}
			tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return response(200), nil
			}), cache)
			ctx, _ := proxy.WithCollector(t.Context())
			target := freshURL("/search")
			for _, body := range []string{"q=one", "q=two"} {
				req, err := http.NewRequestWithContext(ctx, tc.method, target, strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				resp, err := tr.RoundTrip(req)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
			if calls != tc.wantCalls {
				t.Errorf("upstream calls = %d, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestFetchHeadAndGetDoNotShareCacheEntries(t *testing.T) {
	cache := &memCache{entries: map[string]dto.CachedResponse{}}
	var bodies []string
	tr := proxy.NewFetchTransport(identitytest.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := response(200)
		if req.Method == http.MethodHead {
			resp.Body = io.NopCloser(strings.NewReader(""))
		}
		return resp, nil
	}), cache)
	ctx, _ := proxy.WithCollector(t.Context())
	target := freshURL("/job/1")
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequestWithContext(ctx, method, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := tr.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		bodies = append(bodies, string(body))
	}
	if diff := cmp.Diff([]string{"", "ok"}, bodies); diff != "" {
		t.Errorf("bodies (-want +got):\n%s", diff)
	}
}
