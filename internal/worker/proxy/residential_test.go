package proxy_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

type fakeProxy struct {
	*httptest.Server
	mu       sync.Mutex
	sessions []string
}

func (p *fakeProxy) hits() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sessions)
}

func newFakeProxy(t *testing.T, respond http.HandlerFunc) *fakeProxy {
	t.Helper()
	p := &fakeProxy{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimPrefix(r.Header.Get("Proxy-Authorization"), "Basic ")
		creds, _ := base64.StdEncoding.DecodeString(auth)
		user, _, _ := strings.Cut(string(creds), ":")
		p.mu.Lock()
		p.sessions = append(p.sessions, user)
		p.mu.Unlock()
		respond(w, r)
	}))
	t.Cleanup(p.Close)
	return p
}

func status(code int, location string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if location != "" {
			w.Header().Set("Location", location)
		}
		w.WriteHeader(code)
		_, _ = io.WriteString(w, "body")
	}
}

type memCache struct {
	mu      sync.Mutex
	entries map[string]dto.CachedResponse
}

func (c *memCache) LookupFetch(_ context.Context, url string) (dto.CachedResponse, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.entries[url]
	return r, ok, nil
}

func (c *memCache) PutFetch(_ context.Context, resp dto.CachedResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[resp.URL] = resp
	return nil
}

func residentialFetcher(t *testing.T, residential *fakeProxy, cache proxy.Cache) http.RoundTripper {
	t.Helper()
	t.Setenv("DECODO_PROXY_URL", "http://res:pw@"+strings.TrimPrefix(residential.URL, "http://"))
	proxy.SetCache(cache)
	t.Cleanup(func() { proxy.SetCache(nil) })
	tr, err := proxy.Fetcher(proxy.Residential, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func httpTarget(t *testing.T) *http.Request {
	t.Helper()
	return newRequest(t, strings.Replace(freshURL("/jobs"), "https://", "http://", 1))
}

func TestResidentialFetcher(t *testing.T) {
	t.Run("a block is retried on three fresh sessions then fails", func(t *testing.T) {
		blocks := map[string]http.HandlerFunc{
			"429":      status(429, ""),
			"999":      status(999, ""),
			"403":      status(403, ""),
			"authwall": status(302, "https://www.linkedin.com/authwall?trk=x"),
			"login":    status(302, "https://linkedin.com/login"),
		}
		for name, block := range blocks {
			t.Run(name, func(t *testing.T) {
				residential := newFakeProxy(t, block)
				tr := residentialFetcher(t, residential, nil)
				resp, err := tr.RoundTrip(httpTarget(t))
				if !errors.Is(err, proxy.ErrBlocked) {
					t.Fatalf("RoundTrip() = %v, %v, want ErrBlocked", resp, err)
				}
				if got := residential.hits(); got != 3 {
					t.Fatalf("residential attempts = %d, want 3", got)
				}
				seen := map[string]bool{}
				for _, s := range residential.sessions {
					if !strings.HasPrefix(s, "res-session-") {
						t.Fatalf("username %q lacks a session ID", s)
					}
					seen[s] = true
				}
				if len(seen) != 3 {
					t.Fatalf("distinct sessions = %d, want 3: %v", len(seen), residential.sessions)
				}
			})
		}
	})

	t.Run("sessions are reused until a block", func(t *testing.T) {
		var calls atomic.Int32
		residential := newFakeProxy(t, func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 3 {
				status(429, "")(w, r)
				return
			}
			status(200, "")(w, r)
		})
		tr := residentialFetcher(t, residential, nil)
		for range 4 {
			resp, err := tr.RoundTrip(httpTarget(t))
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		s := residential.sessions
		if len(s) != 5 {
			t.Fatalf("residential hits = %d, want 5: %v", len(s), s)
		}
		if s[0] != s[1] || s[1] != s[2] {
			t.Fatalf("sessions before the block differ: %v", s)
		}
		if s[3] == s[2] || s[3] != s[4] {
			t.Fatalf("sessions after the block = %v, want a new one reused", s)
		}
	})

	t.Run("residential 404 is gone and not retried", func(t *testing.T) {
		residential := newFakeProxy(t, status(404, ""))
		resp, err := residentialFetcher(t, residential, nil).RoundTrip(httpTarget(t))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 404 || residential.hits() != 1 {
			t.Fatalf("status=%d residential=%d, want 404/1", resp.StatusCode, residential.hits())
		}
	})

	t.Run("a residential 200 is cached and replayed", func(t *testing.T) {
		residential := newFakeProxy(t, status(200, ""))
		cache := &memCache{entries: map[string]dto.CachedResponse{}}
		tr := residentialFetcher(t, residential, cache)
		ctx, _ := proxy.WithCollector(t.Context())
		req := httpTarget(t).WithContext(ctx)
		for range 2 {
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if string(body) != "body" {
				t.Fatalf("body = %q, want body", body)
			}
		}
		if residential.hits() != 1 {
			t.Fatalf("residential=%d, want 1", residential.hits())
		}
	})
}

func TestResidentialFetcherCachesOnlySuccess(t *testing.T) {
	tests := []struct {
		name       string
		respond    http.HandlerFunc
		wantCached bool
	}{
		{"200", status(200, ""), true},
		{"redirect", status(302, "https://example.com/next"), true},
		{"500", status(500, ""), false},
		{"403", status(403, ""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &memCache{entries: map[string]dto.CachedResponse{}}
			tr := residentialFetcher(t, newFakeProxy(t, tt.respond), cache)
			ctx, _ := proxy.WithCollector(t.Context())
			resp, err := tr.RoundTrip(httpTarget(t).WithContext(ctx))
			if err == nil {
				_ = resp.Body.Close()
			}
			if got := len(cache.entries) == 1; got != tt.wantCached {
				t.Errorf("cached = %v, want %v", got, tt.wantCached)
			}
		})
	}
}

func TestResidentialFetcherLogsRotations(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	residential := newFakeProxy(t, status(429, ""))
	if _, err := residentialFetcher(t, residential, nil).RoundTrip(httpTarget(t)); !errors.Is(err, proxy.ErrBlocked) {
		t.Fatalf("RoundTrip() error = %v, want ErrBlocked", err)
	}

	var counts []float64
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatal(err)
		}
		if rec["msg"] != "residential session rotated" {
			continue
		}
		if rec["source"] != t.Name() || rec["reason"] != "blocked" || rec["status"] != float64(429) {
			t.Errorf("rotation log = %v, want source, reason blocked and status 429", rec)
		}
		counts = append(counts, rec["count"].(float64))
	}
	if len(counts) != 3 || counts[0] != 1 || counts[2] != 3 {
		t.Errorf("rotation counts = %v, want 1..3", counts)
	}
}

func TestValidateResidential(t *testing.T) {
	t.Setenv("DECODO_PROXY_URL", "")
	if err := proxy.ValidateResidential(); err == nil {
		t.Fatal("ValidateResidential() = nil with DECODO_PROXY_URL unset, want error")
	}
	t.Setenv("DECODO_PROXY_URL", "http://gate.decodo.com:7000")
	if err := proxy.ValidateResidential(); err == nil {
		t.Fatal("ValidateResidential() = nil without credentials, want error")
	}
	t.Setenv("DECODO_PROXY_URL", "http://u:p@gate.decodo.com:7000")
	if err := proxy.ValidateResidential(); err != nil {
		t.Fatalf("ValidateResidential() = %v, want nil", err)
	}
}

func TestResidentialFetcherMetrics(t *testing.T) {
	requests := func(source, route, outcome string) float64 {
		return testutil.ToFloat64(proxy.FetchRequests.WithLabelValues(source, route, outcome))
	}
	t.Run("blocked attempts are counted per attempt", func(t *testing.T) {
		residential := newFakeProxy(t, status(429, ""))
		if _, err := residentialFetcher(t, residential, nil).RoundTrip(httpTarget(t)); !errors.Is(err, proxy.ErrBlocked) {
			t.Fatalf("RoundTrip() error = %v, want ErrBlocked", err)
		}
		if got := requests(t.Name(), "residential", "blocked"); got != 3 {
			t.Errorf("residential blocked = %v, want 3", got)
		}
	})
	t.Run("a gone page is counted gone", func(t *testing.T) {
		residential := newFakeProxy(t, status(404, ""))
		resp, err := residentialFetcher(t, residential, nil).RoundTrip(httpTarget(t))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if got := requests(t.Name(), "residential", "gone"); got != 1 {
			t.Errorf("residential gone = %v, want 1", got)
		}
	})
}
