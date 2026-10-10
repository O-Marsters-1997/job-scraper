package indeed_test

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func newScraper(t *testing.T, body string, filters map[string]string, recency ...string) (*indeed.Scraper, *sourcetest.Responder) {
	t.Helper()
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@localhost:7000")
	t.Setenv("INDEED_API_KEY", "test-key")
	s := indeed.New("go developer", filters, append(recency, "")[0])
	r := sourcetest.Respond(body)
	s.Client().Transport = r
	return s, r
}

func TestFetchPage(t *testing.T) {
	fixture, err := os.ReadFile("snapshots/search_london.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	t.Run("maps a card to a complete Job", func(t *testing.T) {
		s, _ := newScraper(t, string(fixture), nil)
		jobs, next, err := s.FetchPage(t.Context(), "")
		if err != nil {
			t.Fatalf("FetchPage() err = %v", err)
		}
		if len(jobs) != 3 {
			t.Fatalf("FetchPage() returned %d jobs, want 3", len(jobs))
		}
		got := jobs[1]
		if got.URL != "https://uk.indeed.com/viewjob?jk=829d375fdc5c3ced" {
			t.Errorf("URL = %q", got.URL)
		}
		if got.Title != "Full Stack Developer" || got.CompanySlug != "orientate" || got.Location != "London" {
			t.Errorf("job = %+v, want title, company slug and location mapped", got)
		}
		if got.SalaryRaw != "£40000 - £80000 per year" {
			t.Errorf("SalaryRaw = %q", got.SalaryRaw)
		}
		if !strings.HasPrefix(got.Description, "<p>Orientate is growing") {
			t.Errorf("Description = %q, want the card's html", got.Description)
		}
		if want := time.UnixMilli(1790945211332).UTC(); !got.UpdatedAt.Equal(want) {
			t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, want)
		}
		if next == "" {
			t.Error("next cursor is empty, want the response's nextCursor")
		}
	})

	t.Run("a card without salary or employer still maps", func(t *testing.T) {
		s, _ := newScraper(t, string(fixture), nil)
		jobs, _, err := s.FetchPage(t.Context(), "")
		if err != nil {
			t.Fatalf("FetchPage() err = %v", err)
		}
		if jobs[0].CompanySlug != "" || jobs[2].SalaryRaw != "" {
			t.Errorf("jobs = %+v, want empty company for the first and empty salary for the last", jobs)
		}
	})

	t.Run("the last page has no next cursor", func(t *testing.T) {
		s, _ := newScraper(t, `{"data":{"jobSearch":{"pageInfo":{"nextCursor":null},"results":[]}}}`, nil)
		jobs, next, err := s.FetchPage(t.Context(), "abc")
		if err != nil || len(jobs) != 0 || next != "" {
			t.Errorf("FetchPage() = %v, %q, %v; want no jobs, no cursor, no error", jobs, next, err)
		}
	})

	t.Run("sends the key, locale headers and the search arguments", func(t *testing.T) {
		filters := map[string]string{"location": `Lon"don`, "radius": "25", "recency": "7"}
		s, r := newScraper(t, string(fixture), filters)
		if _, _, err := s.FetchPage(t.Context(), "cur"); err != nil {
			t.Fatalf("FetchPage() err = %v", err)
		}
		req := r.Last
		headers := map[string]string{"Indeed-Api-Key": "test-key", "Indeed-Co": "GB", "Indeed-Locale": "en-GB"}
		for k, want := range headers {
			if got := req.Header.Get(k); got != want {
				t.Errorf("header %s = %q, want %q", k, got, want)
			}
		}
		body, _ := io.ReadAll(req.Body)
		for _, want := range []string{
			`what: \"go developer\"`,
			`location: {where: \"Lon\\\"don\", radius: 25, radiusUnit: MILES}`,
			`start: \"168h\"`,
			`cursor: \"cur\"`,
		} {
			if !strings.Contains(string(body), want) {
				t.Errorf("request body missing %s:\n%s", want, body)
			}
		}
	})

	t.Run("a missing key is an error", func(t *testing.T) {
		s, _ := newScraper(t, string(fixture), nil)
		t.Setenv("INDEED_API_KEY", "")
		if _, _, err := s.FetchPage(t.Context(), ""); err == nil {
			t.Error("FetchPage() err = nil, want a missing-key error")
		}
	})

	t.Run("a graphql error is an error", func(t *testing.T) {
		s, _ := newScraper(t, `{"errors":[{"message":"bad query"}],"data":null}`, nil)
		_, _, err := s.FetchPage(t.Context(), "")
		if err == nil || !strings.Contains(err.Error(), "bad query") {
			t.Errorf("FetchPage() err = %v, want the graphql message", err)
		}
	})
}

func TestFetchPageKeyRejection(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantReject bool
	}{
		{"401 is a rejected key", http.StatusUnauthorized, "", true},
		{"403 is a rejected key", http.StatusForbidden, "", true},
		{"graphql auth error is a rejected key", http.StatusOK, `{"errors":[{"message":"bad key","extensions":{"code":"UNAUTHENTICATED"}}]}`, true},
		{"500 is not a rejected key", http.StatusInternalServerError, "", false},
		{"429 is not a rejected key", http.StatusTooManyRequests, "", false},
		{"other graphql error is not a rejected key", http.StatusOK, `{"errors":[{"message":"bad query","extensions":{"code":"BAD_USER_INPUT"}}]}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newScraper(t, "", nil)
			s.Client().Transport = sourcetest.RespondStatus(tt.status, tt.body)
			_, _, err := s.FetchPage(t.Context(), "")
			if err == nil {
				t.Fatal("FetchPage() err = nil, want error")
			}
			if got := errors.Is(err, sources.ErrSourceKeyRejected); got != tt.wantReject {
				t.Errorf("FetchPage() errors.Is(ErrSourceKeyRejected) = %v, want %v (err = %v)", got, tt.wantReject, err)
			}
		})
	}
}

func TestRecency(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time {
		at := now.Add(-d)
		return &at
	}
	tests := []struct {
		name       string
		configured string
		last       *time.Time
		want       string
	}{
		{"no previous success sends nothing", "7", nil, ""},
		{"sub-day gap is one day", "7", ago(3 * time.Hour), "1"},
		{"just ran is one day", "7", ago(0), "1"},
		{"multi-day gap rounds up", "14", ago(50 * time.Hour), "3"},
		{"margin pushes an exact day over", "14", ago(24 * time.Hour), "2"},
		{"gap beyond the Target recency keeps stored recency", "3", ago(96 * time.Hour), ""},
		{"gap equal to the Target recency keeps stored recency", "3", ago(72 * time.Hour), ""},
		{"no configured recency still narrows", "", ago(48 * time.Hour), "3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indeed.Recency(tt.configured, tt.last, now); got != tt.want {
				t.Errorf("Recency(%q) = %q, want %q", tt.configured, got, tt.want)
			}
		})
	}
}

func TestIncrementalQuery(t *testing.T) {
	s, r := newScraper(t, `{"data":{"jobSearch":{"pageInfo":{"nextCursor":null},"results":[]}}}`, map[string]string{"recency": "14"}, "3")
	if _, _, err := s.FetchPage(t.Context(), ""); err != nil {
		t.Fatalf("FetchPage() err = %v", err)
	}
	body, _ := io.ReadAll(r.Last.Body)
	for _, want := range []string{`sort: DATE`, `start: \"72h\"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("request body missing %s:\n%s", want, body)
		}
	}
	if !s.Cfg().NewestFirst {
		t.Error("Cfg().NewestFirst = false, want true after a first success")
	}
	first, _ := newScraper(t, "", nil)
	if first.Cfg().NewestFirst {
		t.Error("Cfg().NewestFirst = true with no prior success, want false")
	}
}
