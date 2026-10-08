package workablesearch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func TestHarvester_Harvest(t *testing.T) {
	pages := map[string][]byte{}
	for token, file := range map[string]string{"": "page1.json", "page2token==": "page2.json"} {
		body, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		pages[token] = body
	}
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.UserAgent()
		body, ok := pages[r.URL.Query().Get("pageToken")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	got, err := fastHarvester(srv).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{
		Boards: []discover.Board{{Source: "workable", Token: "acme-direct"}},
		Companies: []discover.Company{
			{Slug: "bask-health", Name: "Bask Health", Board: discover.Board{Source: "workable", Token: "bask-health"}},
			{Slug: "orchard", Name: "Orchard", Board: discover.Board{Source: "workable", Token: "orchard"}},
			{Slug: "pact-coffee", Name: "Pact Coffee", Board: discover.Board{Source: "workable", Token: "pact-coffee"}},
		},
		Skipped: 1,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Harvest (-want +got):\n%s", diff)
	}
	if gotUA != userAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, userAgent)
	}
}

func fastHarvester(srv *httptest.Server) *Harvester {
	h := New(srv.Client(), srv.URL+"/api/v1/jobs?location=United%20Kingdom")
	h.pageGap, h.backoff = 0, time.Millisecond
	return h
}

func rateLimiting(t *testing.T, limited func(token string, hits int) bool) *httptest.Server {
	t.Helper()
	pages := map[string]string{"": "page1.json", "page2token==": "page2.json"}
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("pageToken")
		hits[token]++
		if limited(token, hits[token]) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		body, err := os.ReadFile(filepath.Join("testdata", pages[token]))
		if err != nil {
			t.Error(err)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHarvester_Harvest429(t *testing.T) {
	t.Run("backs off and resumes after a 429", func(t *testing.T) {
		srv := rateLimiting(t, func(token string, hits int) bool { return token == "page2token==" && hits == 1 })
		got, err := fastHarvester(srv).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Companies) != 3 {
			t.Errorf("companies = %d, want all 3 from both pages", len(got.Companies))
		}
	})

	t.Run("keeps the pages it has when the limit never lifts", func(t *testing.T) {
		srv := rateLimiting(t, func(token string, _ int) bool { return token == "page2token==" })
		got, err := fastHarvester(srv).Harvest(t.Context())
		if err != nil {
			t.Fatalf("Harvest() = %v, want the first page kept", err)
		}
		if len(got.Boards)+len(got.Companies) == 0 {
			t.Error("Harvest() kept nothing from the first page")
		}
	})

	t.Run("fails when the first page is limited", func(t *testing.T) {
		srv := rateLimiting(t, func(string, int) bool { return true })
		_, err := fastHarvester(srv).Harvest(t.Context())
		var status *sources.StatusError
		if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests {
			t.Errorf("Harvest() = %v, want a 429 StatusError", err)
		}
	})
}
