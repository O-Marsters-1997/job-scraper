package proxy_test

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

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

func tieredFetcher(t *testing.T, residential, unlocker *fakeProxy, cache proxy.Cache) http.RoundTripper {
	t.Helper()
	t.Setenv("DECODO_PROXY_URL", "http://res:pw@"+strings.TrimPrefix(residential.URL, "http://"))
	t.Setenv("BRIGHTDATA_PROXY_URL", "http://unl:pw@"+strings.TrimPrefix(unlocker.URL, "http://"))
	t.Setenv("BRIGHTDATA_CA_CERT", "")
	proxy.SetCache(cache)
	t.Cleanup(func() { proxy.SetCache(nil) })
	tr, err := proxy.Fetcher(proxy.Tiered)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func httpTarget(t *testing.T) *http.Request {
	t.Helper()
	return newRequest(t, strings.Replace(freshURL("/jobs"), "https://", "http://", 1))
}

func TestTieredFetcher(t *testing.T) {
	t.Run("block retries three residential sessions then one unlocker attempt", func(t *testing.T) {
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
				unlocker := newFakeProxy(t, status(200, ""))
				tr := tieredFetcher(t, residential, unlocker, nil)
				resp, err := tr.RoundTrip(httpTarget(t))
				if err != nil {
					t.Fatal(err)
				}
				_ = resp.Body.Close()
				if resp.StatusCode != 200 {
					t.Fatalf("status = %d, want 200 from unlocker", resp.StatusCode)
				}
				if got := residential.hits(); got != 3 {
					t.Fatalf("residential attempts = %d, want 3", got)
				}
				if got := unlocker.hits(); got != 1 {
					t.Fatalf("unlocker attempts = %d, want 1", got)
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

	t.Run("residential 404 is gone without fallback", func(t *testing.T) {
		residential := newFakeProxy(t, status(404, ""))
		unlocker := newFakeProxy(t, status(200, ""))
		resp, err := tieredFetcher(t, residential, unlocker, nil).RoundTrip(httpTarget(t))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 404 || residential.hits() != 1 || unlocker.hits() != 0 {
			t.Fatalf("status=%d residential=%d unlocker=%d, want 404/1/0", resp.StatusCode, residential.hits(), unlocker.hits())
		}
	})

	t.Run("cached 200 skips both tiers and a residential 200 is cached", func(t *testing.T) {
		residential := newFakeProxy(t, status(200, ""))
		unlocker := newFakeProxy(t, status(200, ""))
		cache := &memCache{entries: map[string]dto.CachedResponse{}}
		tr := tieredFetcher(t, residential, unlocker, cache)
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
		if residential.hits() != 1 || unlocker.hits() != 0 {
			t.Fatalf("residential=%d unlocker=%d, want 1/0", residential.hits(), unlocker.hits())
		}
	})
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
