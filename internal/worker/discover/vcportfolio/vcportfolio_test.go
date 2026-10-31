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

const gone = "http://127.0.0.1:1"

func serve(t *testing.T, pages map[string]string) string {
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
	return srv.URL
}

func harvest(t *testing.T, sources vcportfolio.Sources, get discover.GetFunc) (discover.Harvest, error) {
	t.Helper()
	return vcportfolio.New(http.DefaultClient, sources, get, slog.New(slog.DiscardHandler)).Harvest(t.Context())
}

func acmeSite(fetched *[]string) discover.GetFunc {
	return func(_ context.Context, url string) ([]byte, error) {
		*fetched = append(*fetched, url)
		if url == "https://acme.co.uk" {
			return []byte(`<a href="https://boards.greenhouse.io/acme">Jobs</a>`), nil
		}
		return nil, fmt.Errorf("unexpected GET %s", url)
	}
}

var acme = discover.Company{Slug: "acme", Name: "Acme", Domain: "acme.co.uk", Board: discover.Board{Source: "greenhouse", Token: "acme"}}

func TestHarvester_Harvest(t *testing.T) {
	t.Parallel()

	t.Run("resolves each company website once across portfolios", func(t *testing.T) {
		t.Parallel()
		vc := serve(t, map[string]string{
			"/a": `<a href="/about">About</a>
				<a href="https://www.acme.co.uk/">Acme</a>
				<a href="https://uk.linkedin.com/company/acme">LinkedIn</a>
				<a href="mailto:hi@vc.example">Mail</a>
				<a href="https://jobs.lever.co/widget">Widget jobs</a>`,
			"/b": `<a href="https://app.acme.co.uk/login">Acme</a>`,
		})
		var fetched []string

		found, err := harvest(t, vcportfolio.Sources{Portfolios: []string{vc + "/a", vc + "/b"}}, acmeSite(&fetched))
		if err != nil {
			t.Fatal(err)
		}

		want := discover.Harvest{
			Boards:    []discover.Board{{Source: "lever", Token: "widget"}},
			Companies: []discover.Company{acme},
		}
		if diff := cmp.Diff(want, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"https://acme.co.uk"}, fetched); diff != "" {
			t.Errorf("Harvest fetched (-want +got):\n%s", diff)
		}
	})

	t.Run("follows a job board sitemap to its company pages", func(t *testing.T) {
		t.Parallel()
		board := serve(t, map[string]string{
			"/sitemap.xml": `<?xml version="1.0" encoding="UTF-8"?>
				<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
					<url><loc>https://talent.vc.example/jobs</loc></url>
					<url><loc>https://talent.vc.example/companies/acme</loc></url>
					<url><loc>https://talent.vc.example/companies/acme/jobs/1-engineer</loc></url>
				</urlset>`,
			"/companies/acme": `<a href="http://acme.co.uk">acme.co.uk</a>
				<a href="https://www.getro.com/vc">Powered by Getro</a>`,
		})
		var fetched []string

		found, err := harvest(t, vcportfolio.Sources{GetroJobBoards: []string{board}}, acmeSite(&fetched))
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(discover.Harvest{Companies: []discover.Company{acme}}, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"https://acme.co.uk"}, fetched); diff != "" {
			t.Errorf("Harvest fetched (-want +got):\n%s", diff)
		}
	})

	t.Run("website with no board is skipped", func(t *testing.T) {
		t.Parallel()
		vc := serve(t, map[string]string{"/a": `<a href="https://acme.io">Acme</a>`})
		get := func(context.Context, string) ([]byte, error) { return []byte(`<p>hello</p>`), nil }

		found, err := harvest(t, vcportfolio.Sources{Portfolios: []string{vc + "/a"}}, get)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(discover.Harvest{Skipped: 1}, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
	})

	t.Run("failing pages and boards keep the rest", func(t *testing.T) {
		t.Parallel()
		vc := serve(t, map[string]string{"/a": `<a href="https://jobs.lever.co/widget">Jobs</a>`})
		get := func(context.Context, string) ([]byte, error) { return nil, fmt.Errorf("no fetch expected") }

		found, err := harvest(t, vcportfolio.Sources{Portfolios: []string{vc + "/a", gone + "/a"}, GetroJobBoards: []string{gone}}, get)
		if err != nil {
			t.Fatal(err)
		}

		want := discover.Harvest{Boards: []discover.Board{{Source: "lever", Token: "widget"}}}
		if diff := cmp.Diff(want, found); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
	})

	t.Run("every source failing is an error", func(t *testing.T) {
		t.Parallel()
		get := func(context.Context, string) ([]byte, error) { return nil, fmt.Errorf("no fetch expected") }

		_, err := harvest(t, vcportfolio.Sources{Portfolios: []string{gone + "/a"}, GetroJobBoards: []string{gone}}, get)
		if err == nil {
			t.Error("Harvest() error = nil, want an error")
		}
	})
}
