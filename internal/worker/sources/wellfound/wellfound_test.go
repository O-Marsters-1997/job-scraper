package wellfound_test

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wellfound"
)

func TestMain(m *testing.M) {
	if err := os.Setenv("DECODO_PROXY_URL", "http://user:pass@localhost:7000"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func newScraper(search wellfound.Search) *wellfound.Scraper {
	return wellfound.New(search, sources.NewGate(0, time.Hour))
}

func TestSnapshots(t *testing.T) {
	t.Parallel()
	sourcetest.RunSnapshotTests(t, newScraper(wellfound.Search{}))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, newScraper(wellfound.Search{}))
}

func TestScraper_FetchPage(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("snapshots/list_role-location.html")
	if err != nil {
		t.Fatal(err)
	}
	search := wellfound.Search{Role: "software-engineer", Location: "london"}
	tests := []struct {
		name     string
		cursor   string
		wantNext string
		wantPath string
	}{
		{"first page asks for page 2 next", "", "2", "/role/l/software-engineer/london"},
		{"a middle page asks for the next", "3", "4", "/role/l/software-engineer/london?page=3"},
		{"the last page ends paging", "5", "", "/role/l/software-engineer/london?page=5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newScraper(search)
			tr := sourcetest.Respond(string(body))
			s.Client().Transport = tr
			jobs, next, err := s.FetchPage(t.Context(), tt.cursor)
			if err != nil {
				t.Fatal(err)
			}
			if len(jobs) == 0 {
				t.Error("FetchPage() returned no jobs")
			}
			if next != tt.wantNext {
				t.Errorf("FetchPage() next = %q, want %q", next, tt.wantNext)
			}
			if got := tr.Last.URL.RequestURI(); got != tt.wantPath {
				t.Errorf("FetchPage() requested %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestScraper_FetchPage_Requests(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("snapshots/list_location.html")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		search   wellfound.Search
		wantPath string
	}{
		{"role only", wellfound.Search{Role: "software-engineer"}, "/role/software-engineer"},
		{"location only", wellfound.Search{Location: "london"}, "/location/london"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newScraper(tt.search)
			tr := sourcetest.Respond(string(body))
			s.Client().Transport = tr
			if _, _, err := s.FetchPage(t.Context(), ""); err != nil {
				t.Fatal(err)
			}
			if got := tr.Last.URL.RequestURI(); got != tt.wantPath {
				t.Errorf("FetchPage() requested %q, want %q", got, tt.wantPath)
			}
		})
	}

	t.Run("an empty search is rejected", func(t *testing.T) {
		t.Parallel()
		if _, _, err := newScraper(wellfound.Search{}).FetchPage(t.Context(), ""); err == nil {
			t.Error("FetchPage() error = nil, want an error")
		}
	})
	t.Run("a bad cursor is rejected", func(t *testing.T) {
		t.Parallel()
		if _, _, err := newScraper(wellfound.Search{Location: "london"}).FetchPage(t.Context(), "abc"); err == nil {
			t.Error("FetchPage() error = nil, want an error")
		}
	})
	t.Run("a challenge page blocks the gate", func(t *testing.T) {
		t.Parallel()
		s := wellfound.New(wellfound.Search{Location: "london"}, sources.NewGate(0, time.Hour))
		s.Client().Transport = sourcetest.Respond("<html>captcha</html>")
		if _, _, err := s.FetchPage(t.Context(), ""); !errors.Is(err, proxy.ErrBlocked) {
			t.Fatalf("FetchPage() error = %v, want ErrBlocked", err)
		}
		if _, _, err := s.FetchPage(t.Context(), ""); !errors.Is(err, sources.ErrDeferred) {
			t.Errorf("second FetchPage() error = %v, want ErrDeferred", err)
		}
	})
}

func TestScraper_ParseURLs(t *testing.T) {
	t.Parallel()
	s := newScraper(wellfound.Search{})
	t.Run("a challenge page is blocked", func(t *testing.T) {
		t.Parallel()
		_, err := s.ParseURLs(bytes.NewBufferString("<html>captcha</html>"))
		if !errors.Is(err, proxy.ErrBlocked) {
			t.Fatalf("ParseURLs() error = %v, want ErrBlocked", err)
		}
	})
}
