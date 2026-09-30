package crawl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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

func seedCompany(t *testing.T, store *jobsearchtest.FakeStore, domain string) dto.Company {
	t.Helper()
	company, err := store.UpsertCompany(context.Background(), dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: domain})
	if err != nil {
		t.Fatal(err)
	}
	return company
}

func assertCrawled(t *testing.T, store *jobsearchtest.FakeStore) {
	t.Helper()
	pending, err := store.ListCompaniesToCrawl(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("pending crawl = %+v, want none: a crawl, hit or miss, touches last_crawled_at", pending)
	}
}

func TestCrawlCompanyResolvesGreenhouseBoard(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /blocked\n"))
	})
	mux.HandleFunc("/careers", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><a href="https://boards.greenhouse.io/acmecorp">Careers</a></body></html>`))
	})

	server := httptest.NewTLSServer(mux)
	defer server.Close()

	host, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	store := jobsearchtest.NewFakeStore()
	company := seedCompany(t, store, host.Host)
	crawler := crawl.New(store).WithClient(server.Client())

	crawler.CrawlCompany(context.Background(), company)

	got, err := store.GetCompany(context.Background(), company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ATSSource != "greenhouse" || got.ATSToken != "acmecorp" {
		t.Errorf("company = %+v, want ats_source=greenhouse ats_token=acmecorp", got)
	}
	if got.Slug != "acme" || got.Name != "Acme" {
		t.Errorf("company = %+v, want slug/name preserved from the existing company row", got)
	}
	assertCrawled(t, store)
}

func TestCrawlCompanyNeverFetchesDisallowedPath(t *testing.T) {
	var careersFetched bool

	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /careers\n"))
	})
	mux.HandleFunc("/careers", func(w http.ResponseWriter, _ *http.Request) {
		careersFetched = true
		_, _ = w.Write([]byte(`<html><body><a href="https://boards.greenhouse.io/acmecorp">Careers</a></body></html>`))
	})
	mux.HandleFunc("/jobs", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body>No board here.</body></html>`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body>No board here either.</body></html>`))
	})

	server := httptest.NewTLSServer(mux)
	defer server.Close()

	host, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	store := jobsearchtest.NewFakeStore()
	company := seedCompany(t, store, host.Host)
	crawler := crawl.New(store).WithClient(server.Client())

	crawler.CrawlCompany(context.Background(), company)

	if careersFetched {
		t.Error("expected /careers to never be fetched: robots.txt disallows it")
	}
	got, err := store.GetCompany(context.Background(), company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ATSSource != "" {
		t.Errorf("expected no ATS resolution, got %+v", got)
	}
	assertCrawled(t, store)
}

func TestCrawlCompanyTreatsUnreachableRobotsAsAllowAll(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/careers", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><a href="https://boards.greenhouse.io/acmecorp">Careers</a></body></html>`))
	})

	server := httptest.NewTLSServer(mux)
	defer server.Close()

	host, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	store := jobsearchtest.NewFakeStore()
	company := seedCompany(t, store, host.Host)
	crawler := crawl.New(store).WithClient(server.Client())

	crawler.CrawlCompany(context.Background(), company)

	got, err := store.GetCompany(context.Background(), company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ATSToken != "acmecorp" {
		t.Fatalf("company = %+v, want the greenhouse board despite the missing robots.txt", got)
	}
}
