package crawl_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/crawl"
)

func TestParseATSLinks(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		base       string
		wantSource string
		wantToken  string
	}{
		{
			name:       "figma careers page embeds greenhouse board",
			fixture:    "testdata/figma_careers_greenhouse.html",
			base:       "https://www.figma.com/careers/",
			wantSource: "greenhouse",
			wantToken:  "figma",
		},
		{
			name:       "notion careers page embeds ashby board",
			fixture:    "testdata/notion_careers_ashby.html",
			base:       "https://www.notion.so/careers",
			wantSource: "ashby",
			wantToken:  "notion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.fixture)
			if err != nil {
				t.Fatalf("open fixture: %v", err)
			}
			defer func() { _ = f.Close() }()

			base, err := url.Parse(tt.base)
			if err != nil {
				t.Fatalf("parse base: %v", err)
			}

			links, err := crawl.ParseATSLinks(f, base)
			if err != nil {
				t.Fatalf("ParseATSLinks: %v", err)
			}
			if len(links) == 0 {
				t.Fatalf("no links found")
			}

			var gotSource, gotToken string
			var ok bool
			for _, link := range links {
				if gotSource, gotToken, ok = detect.ResolveBoard(link); ok {
					break
				}
			}
			if !ok {
				t.Fatalf("no link out of %d resolved to an ATS board", len(links))
			}
			if gotSource != tt.wantSource || gotToken != tt.wantToken {
				t.Errorf("resolved (%q, %q), want (%q, %q)", gotSource, gotToken, tt.wantSource, tt.wantToken)
			}
		})
	}
}

const greenhouseLink = `<html><body><a href="https://boards.greenhouse.io/acmecorp">Careers</a></body></html>`

func serve(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }
}

func crawlSite(t *testing.T, routes map[string]http.HandlerFunc) (*jobsearchtest.FakeStore, dto.Company) {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range routes {
		mux.HandleFunc(path, h)
	}
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	host, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	store := jobsearchtest.NewFakeStore()
	company, err := store.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: host.Host})
	if err != nil {
		t.Fatal(err)
	}
	crawl.New(store).WithClient(server.Client()).CrawlCompany(t.Context(), company)
	got, err := store.GetCompany(t.Context(), company.ID)
	if err != nil {
		t.Fatal(err)
	}
	return store, got
}

func assertCrawled(t *testing.T, store *jobsearchtest.FakeStore) {
	t.Helper()
	pending, err := store.ListCompaniesToCrawl(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("pending crawl = %+v, want none: a crawl, hit or miss, touches last_crawled_at", pending)
	}
}

func TestCrawlCompanyResolvesGreenhouseBoard(t *testing.T) {
	store, got := crawlSite(t, map[string]http.HandlerFunc{
		"/robots.txt": serve("User-agent: *\nDisallow: /blocked\n"),
		"/careers":    serve(greenhouseLink),
	})
	if got.ATSSource != "greenhouse" || got.ATSToken != "acmecorp" {
		t.Errorf("company = %+v, want ats_source=greenhouse ats_token=acmecorp", got)
	}
	if got.Slug != "acme" || got.Name != "Acme" {
		t.Errorf("company = %+v, want slug/name preserved from the existing company row", got)
	}
	assertCrawled(t, store)
}

func TestCrawlCompanyNeverFetchesDisallowedPath(t *testing.T) {
	var careersFetched atomic.Bool
	store, got := crawlSite(t, map[string]http.HandlerFunc{
		"/robots.txt": serve("User-agent: *\nDisallow: /careers\n"),
		"/careers": func(w http.ResponseWriter, r *http.Request) {
			careersFetched.Store(true)
			serve(greenhouseLink)(w, r)
		},
		"/jobs": serve("<html><body>No board here.</body></html>"),
		"/":     serve("<html><body>No board here either.</body></html>"),
	})
	if careersFetched.Load() {
		t.Error("expected /careers to never be fetched: robots.txt disallows it")
	}
	if got.ATSSource != "" {
		t.Errorf("expected no ATS resolution, got %+v", got)
	}
	assertCrawled(t, store)
}

func TestCrawlCompanyTreatsUnreachableRobotsAsAllowAll(t *testing.T) {
	_, got := crawlSite(t, map[string]http.HandlerFunc{"/careers": serve(greenhouseLink)})
	if got.ATSToken != "acmecorp" {
		t.Fatalf("company = %+v, want the greenhouse board despite the missing robots.txt", got)
	}
}
