package wttj_test

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

var update = flag.Bool("update", false, "rewrite golden files from parser output")

var now = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

type site struct {
	mu       sync.Mutex
	company  string
	jobPages map[string]string
	status   int
	requests []string
}

func (s *site) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req.URL.Path)
	respond := func(status int, body string) (*http.Response, error) {
		return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	if s.status != 0 {
		return respond(s.status, "")
	}
	if strings.HasPrefix(req.URL.Path, "/companies/") {
		return respond(http.StatusOK, s.company)
	}
	if page, ok := s.jobPages[strings.TrimPrefix(req.URL.Path, "/jobs/")]; ok {
		return respond(http.StatusOK, page)
	}
	return respond(http.StatusNotFound, "")
}

func (s *site) jobRequests() int {
	n := 0
	for _, p := range s.requests {
		if strings.HasPrefix(p, "/jobs/") {
			n++
		}
	}
	return n
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("snapshots", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(body)
}

func newSite(t *testing.T, company string) *site {
	t.Helper()
	return &site{
		company: fixture(t, company),
		jobPages: map[string]string{
			"rj-ZQt7d": fixture(t, "job_ashby.html"),
			"a-1anGDH": fixture(t, "job_greenhouse.html"),
			"8-Y5adJl": fixture(t, "job_workday.html"),
		},
	}
}

func poll(t *testing.T, token string, s *site) ([]dto.Job, error) {
	t.Helper()
	src := wttj.New(token, func() time.Time { return now })
	src.Client().Transport = s
	jobs, _, err := src.FetchPage(t.Context(), "")
	return jobs, err
}

func assertGolden(t *testing.T, name string, got []dto.Job) {
	t.Helper()
	path := filepath.Join("snapshots", name)
	if *update {
		out, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v (run `go test -update`)", err)
	}
	var want []dto.Job
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FetchPage mismatch (-want +got):\n%s", diff)
	}
}

func TestFetchPage(t *testing.T) {
	t.Run("UK company returns jobs posted within 90 days with datePosted as UpdatedAt", func(t *testing.T) {
		jobs, err := poll(t, "faculty-golden", newSite(t, "company_faculty.html"))
		if err != nil {
			t.Fatal(err)
		}
		assertGolden(t, "company_faculty.golden.json", jobs)
	})

	t.Run("US-only company fetches no job pages", func(t *testing.T) {
		s := newSite(t, "company_us_only.html")
		jobs, err := poll(t, "us-only", s)
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 0 || s.jobRequests() != 0 {
			t.Errorf("FetchPage = %d jobs, %d job pages, want none", len(jobs), s.jobRequests())
		}
	})

	t.Run("company with no live jobs returns none", func(t *testing.T) {
		jobs, err := poll(t, "no-jobs", newSite(t, "company_no_jobs.html"))
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 0 {
			t.Errorf("FetchPage = %d jobs, want 0", len(jobs))
		}
	})

	t.Run("second poll fetches no job page already read", func(t *testing.T) {
		s := newSite(t, "company_faculty.html")
		first, err := poll(t, "faculty-repeat", s)
		if err != nil {
			t.Fatal(err)
		}
		fetched := s.jobRequests()
		second, err := poll(t, "faculty-repeat", s)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.jobRequests(); got != fetched {
			t.Errorf("job pages fetched on second poll = %d, want 0", got-fetched)
		}
		if diff := cmp.Diff(first, second); diff != "" {
			t.Errorf("second poll jobs differ (-first +second):\n%s", diff)
		}
	})

	t.Run("403 and 429 fail the poll without retrying", func(t *testing.T) {
		for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests} {
			s := newSite(t, "company_faculty.html")
			s.status = status
			if _, err := poll(t, "blocked", s); err == nil {
				t.Errorf("FetchPage with status %d error = %v, want error", status, err)
			}
			if len(s.requests) != 1 {
				t.Errorf("status %d: %d requests, want 1", status, len(s.requests))
			}
		}
	})
}
