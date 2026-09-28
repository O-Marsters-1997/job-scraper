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

type fakeStore struct {
	toCrawl []dto.Company
	upserts []dto.CompanyUpsert
	touched []string
}

func (f *fakeStore) ListCompaniesToCrawl(_ context.Context, limit int) ([]dto.Company, error) {
	if len(f.toCrawl) > limit {
		return f.toCrawl[:limit], nil
	}
	return f.toCrawl, nil
}

func (f *fakeStore) TouchCompanyCrawled(_ context.Context, id string) error {
	f.touched = append(f.touched, id)
	return nil
}

func (f *fakeStore) UpsertCompany(_ context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	f.upserts = append(f.upserts, c)
	return dto.Company{Slug: c.Slug, Name: c.Name, ATSSource: c.ATSSource, ATSToken: c.ATSToken, Domain: c.Domain}, nil
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

	store := &fakeStore{toCrawl: []dto.Company{
		{ID: "co-1", Slug: "acme", Name: "Acme", Domain: host.Host},
	}}
	crawler := crawl.New(store).WithClient(server.Client())

	crawler.CrawlCompany(context.Background(), store.toCrawl[0])

	if len(store.upserts) != 1 {
		t.Fatalf("expected 1 upsert, got %d", len(store.upserts))
	}
	got := store.upserts[0]
	if got.ATSSource != "greenhouse" || got.ATSToken != "acmecorp" {
		t.Errorf("upsert = %+v, want ats_source=greenhouse ats_token=acmecorp", got)
	}
	if got.Slug != "acme" || got.Name != "Acme" {
		t.Errorf("upsert = %+v, want slug/name preserved from the existing company row", got)
	}
	if len(store.touched) != 1 || store.touched[0] != "co-1" {
		t.Errorf("touched = %v, want [co-1]", store.touched)
	}
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

	store := &fakeStore{toCrawl: []dto.Company{
		{ID: "co-2", Slug: "blocked-co", Name: "Blocked Co", Domain: host.Host},
	}}
	crawler := crawl.New(store).WithClient(server.Client())

	crawler.CrawlCompany(context.Background(), store.toCrawl[0])

	if careersFetched {
		t.Error("expected /careers to never be fetched: robots.txt disallows it")
	}
	if len(store.upserts) != 0 {
		t.Errorf("expected no ATS resolution, got upserts: %+v", store.upserts)
	}
	if len(store.touched) != 1 || store.touched[0] != "co-2" {
		t.Errorf("touched = %v, want [co-2] (a miss still touches last_crawled_at)", store.touched)
	}
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

	store := &fakeStore{toCrawl: []dto.Company{
		{ID: "co-3", Slug: "acme", Name: "Acme", Domain: host.Host},
	}}
	crawler := crawl.New(store).WithClient(server.Client())

	crawler.CrawlCompany(context.Background(), store.toCrawl[0])

	if len(store.upserts) != 1 || store.upserts[0].ATSToken != "acmecorp" {
		t.Fatalf("upserts = %+v, want the greenhouse board despite the missing robots.txt", store.upserts)
	}
}
