package jobsearch_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

type scriptedQuota struct {
	quota jobsearch.ProxyQuota
	err   error
}

func (s *scriptedQuota) FetchQuota(context.Context) (jobsearch.ProxyQuota, error) {
	return s.quota, s.err
}

func newUsageRouter(t *testing.T, fetcher jobsearch.QuotaFetcher, now time.Time) http.Handler {
	t.Helper()
	deps := jobsearchtest.NewDeps(jobsearchtest.NewFakeStore())
	deps.ProxyQuota = fetcher
	deps.Now = func() time.Time { return now }
	m := jobsearch.Build(deps)
	if err := m.RefreshProxyUsage(t.Context()); err != nil {
		t.Fatalf("RefreshProxyUsage() = %v", err)
	}
	r := chi.NewRouter()
	m.Routes(r)
	return r
}

func getProxyUsage(t *testing.T, h http.Handler, role string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/usage/proxies", http.NoBody)
	req = req.WithContext(handlers.WithSession(req.Context(), dto.Session{UserID: handlerstest.UserID, Role: role}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestProxyUsageRoute(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	resets := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	t.Run("a non-admin gets 403", func(t *testing.T) {
		h := newUsageRouter(t, nil, now)
		if w := getProxyUsage(t, h, dto.RoleUser); w.Code != http.StatusForbidden {
			t.Errorf("GET /usage/proxies as user = %d, want 403", w.Code)
		}
	})

	t.Run("an unset key reports not configured and makes no call", func(t *testing.T) {
		h := newUsageRouter(t, nil, now)
		w := getProxyUsage(t, h, dto.RoleAdmin)
		got := handlerstest.DecodeJSON[jobsearch.ProxyUsageView](t, w.Body.Bytes())
		want := jobsearch.ProxyUsageView{Providers: []dto.QuotaView{
			{Provider: "decodo", Status: dto.QuotaNotConfigured, Unit: "GB", Level: dto.UsageOK},
		}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("view (-want +got):\n%s", diff)
		}
	})

	t.Run("a good fetch reports used, limit, level and reset", func(t *testing.T) {
		f := &scriptedQuota{quota: jobsearch.ProxyQuota{Used: 9, Limit: 10, ResetsAt: resets}}
		h := newUsageRouter(t, f, now)
		w := getProxyUsage(t, h, dto.RoleAdmin)
		got := handlerstest.DecodeJSON[jobsearch.ProxyUsageView](t, w.Body.Bytes()).Providers[0]
		if got.Status != dto.QuotaOK || got.Level != dto.UsageWarn || *got.Used != 9 || *got.Limit != 10 || !got.ResetsAt.Equal(resets) || !got.FetchedAt.Equal(now) {
			t.Errorf("view = %+v, want ok/warn 9 of 10 resetting %v fetched %v", got, resets, now)
		}
	})
}

func TestProxyUsageRefresh(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	f := &scriptedQuota{quota: jobsearch.ProxyQuota{Used: 2, Limit: 10, ResetsAt: now.Add(time.Hour)}}
	deps := jobsearchtest.NewDeps(jobsearchtest.NewFakeStore())
	deps.ProxyQuota = f
	deps.Now = func() time.Time { return now }
	m := jobsearch.Build(deps)
	r := chi.NewRouter()
	m.Routes(r)

	if err := m.RefreshProxyUsage(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.err = errors.New("boom")
	if err := m.RefreshProxyUsage(t.Context()); err != nil {
		t.Fatalf("RefreshProxyUsage() on fetch failure = %v, want nil", err)
	}

	w := getProxyUsage(t, r, dto.RoleAdmin)
	got := handlerstest.DecodeJSON[jobsearch.ProxyUsageView](t, w.Body.Bytes()).Providers[0]
	if got.Status != dto.QuotaError || got.Error != "boom" {
		t.Errorf("status/error = %q/%q, want error/boom", got.Status, got.Error)
	}
	if got.Used == nil || *got.Used != 2 || got.FetchedAt == nil {
		t.Errorf("view = %+v, want last good numbers and FetchedAt kept", got)
	}
}

func TestNewDecodoQuota(t *testing.T) {
	serve := func(status int, body string) (*httptest.Server, *string) {
		var auth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv, &auth
	}

	t.Run("parses the string fields and sends the key", func(t *testing.T) {
		srv, auth := serve(http.StatusOK, `[{"traffic_per_period":"1.5","traffic_limit":"8","valid_until":"2026-11-01"}]`)
		got, err := jobsearch.NewDecodoQuota(srv.Client(), srv.URL, "k").FetchQuota(t.Context())
		if err != nil {
			t.Fatalf("FetchQuota() = %v", err)
		}
		want := jobsearch.ProxyQuota{Used: 1.5, Limit: 8, ResetsAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("quota (-want +got):\n%s", diff)
		}
		if *auth != "Basic k" {
			t.Errorf("Authorization = %q, want %q", *auth, "Basic k")
		}
	})

	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"non-200":       {http.StatusUnauthorized, `{}`},
		"empty list":    {http.StatusOK, `[]`},
		"bad number":    {http.StatusOK, `[{"traffic_per_period":"x","traffic_limit":"8","valid_until":"2026-11-01"}]`},
		"bad timestamp": {http.StatusOK, `[{"traffic_per_period":"1","traffic_limit":"8","valid_until":"soon"}]`},
	} {
		t.Run(name+" is an error", func(t *testing.T) {
			srv, _ := serve(tc.status, tc.body)
			if _, err := jobsearch.NewDecodoQuota(srv.Client(), srv.URL, "k").FetchQuota(t.Context()); err == nil {
				t.Error("FetchQuota() = nil error, want one")
			}
		})
	}
}
