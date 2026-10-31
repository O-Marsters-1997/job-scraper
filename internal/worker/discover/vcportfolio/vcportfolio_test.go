package vcportfolio_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/vcportfolio"
)

func portfolio(t *testing.T, pages map[string]string) []string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	urls := make([]string, 0, len(pages))
	for path := range pages {
		urls = append(urls, srv.URL+path)
	}
	return urls
}

func TestHarvester_Harvest(t *testing.T) {
	t.Parallel()
	t.Run("resolves each company website once across portfolios", func(t *testing.T) {
		t.Parallel()
		pages := portfolio(t, map[string]string{
			"/a": `<a href="/about">About</a>
				<a href="https://www.acme.co.uk/">Acme</a>
				<a href="https://uk.linkedin.com/company/acme">LinkedIn</a>
				<a href="mailto:hi@vc.example">Mail</a>
				<a href="https://jobs.lever.co/widget">Widget jobs</a>`,
			"/b": `<a href="https://app.acme.co.uk/login">Acme</a>`,
		})
		var fetched []string
		get := func(_ context.Context, url string) ([]byte, error) {
			fetched = append(fetched, url)
			if url == "https://acme.co.uk" {
				return []byte(`<a href="https://boards.greenhouse.io/acme">Jobs</a>`), nil
			}
			return nil, fmt.Errorf("unexpected GET %s", url)
		}

		found, err := vcportfolio.New(http.DefaultClient, pages, get, slog.New(slog.DiscardHandler)).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		want := discover.Harvest{
			Boards: []discover.Board{{Source: "lever", Token: "widget"}},
			Companies: []discover.Company{
				{Slug: "acme", Name: "Acme", Domain: "acme.co.uk", Board: discover.Board{Source: "greenhouse", Token: "acme"}},
			},
		}
		if diff := cmp.Diff(want, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"https://acme.co.uk"}, fetched); diff != "" {
			t.Errorf("Harvest fetched (-want +got):\n%s", diff)
		}
	})

	t.Run("website with no board is skipped", func(t *testing.T) {
		t.Parallel()
		pages := portfolio(t, map[string]string{"/a": `<a href="https://acme.io">Acme</a>`})
		get := func(context.Context, string) ([]byte, error) { return []byte(`<p>hello</p>`), nil }

		found, err := vcportfolio.New(http.DefaultClient, pages, get, slog.New(slog.DiscardHandler)).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(discover.Harvest{Skipped: 1}, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
	})

	t.Run("one failing page keeps the others", func(t *testing.T) {
		t.Parallel()
		pages := append(portfolio(t, map[string]string{"/a": `<a href="https://jobs.lever.co/widget">Jobs</a>`}), "http://127.0.0.1:1/gone")
		get := func(context.Context, string) ([]byte, error) { return nil, fmt.Errorf("no fetch expected") }

		found, err := vcportfolio.New(http.DefaultClient, pages, get, slog.New(slog.DiscardHandler)).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		want := discover.Harvest{Boards: []discover.Board{{Source: "lever", Token: "widget"}}}
		if diff := cmp.Diff(want, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
	})

	t.Run("every page failing is an error", func(t *testing.T) {
		t.Parallel()
		get := func(context.Context, string) ([]byte, error) { return nil, fmt.Errorf("no fetch expected") }

		_, err := vcportfolio.New(http.DefaultClient, []string{"http://127.0.0.1:1/gone"}, get, slog.New(slog.DiscardHandler)).Harvest(t.Context())
		if err == nil {
			t.Error("Harvest() error = nil, want an error")
		}
	})
}
