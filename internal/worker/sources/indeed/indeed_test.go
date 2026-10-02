package indeed_test

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func newScraper(t *testing.T, body string, filters map[string]string) (*indeed.Scraper, *sourcetest.Responder) {
	t.Helper()
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@localhost:7000")
	t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@localhost:7001")
	t.Setenv("INDEED_API_KEY", "test-key")
	s := indeed.New("go developer", filters)
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
