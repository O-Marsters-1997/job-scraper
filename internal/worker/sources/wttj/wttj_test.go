package wttj_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

var now = time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)

type api struct {
	mu         sync.Mutex
	company    string
	jobs       string
	status     int
	retryAfter string
	queries    []string
}

func (a *api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.queries = append(a.queries, body.Query)
	if a.status != 0 {
		if a.retryAfter != "" {
			w.Header().Set("Retry-After", a.retryAfter)
		}
		w.WriteHeader(a.status)
		return
	}
	if strings.Contains(body.Query, "company(") {
		_, _ = w.Write([]byte(a.company))
		return
	}
	_, _ = w.Write([]byte(a.jobs))
}

func (a *api) jobQueries() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for _, q := range a.queries {
		if strings.Contains(q, "publicJob") {
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

func (a *api) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, r)
	return rec.Result(), nil
}

func newAPI(t *testing.T, company string) (*api, *wttj.Client) {
	t.Helper()
	a := &api{company: fixture(t, company), jobs: fixture(t, "jobs_faculty.json")}
	client := wttj.NewClient(0)
	client.Client().Transport = a
	return a, client
}

func source(client *wttj.Client) *wttj.Source {
	return wttj.New("Faculty", func() time.Time { return now }, client)
}

func TestFetchPage(t *testing.T) {
	t.Parallel()
	t.Run("UK company returns its UK engineering jobs posted within 90 days", func(t *testing.T) {
		t.Parallel()
		_, client := newAPI(t, "company_faculty.json")
		jobs, _, err := source(client).FetchPage(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		sourcetest.MatchGolden(t, "company_faculty.json", jobs)
	})

	t.Run("US-only company looks up no jobs", func(t *testing.T) {
		t.Parallel()
		a, client := newAPI(t, "company_us_only.json")
		jobs, _, err := source(client).FetchPage(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 0 || a.jobQueries() != 0 {
			t.Errorf("FetchPage = %d jobs, %d job queries, want none", len(jobs), a.jobQueries())
		}
	})

	t.Run("company with no live jobs looks up none", func(t *testing.T) {
		t.Parallel()
		a, client := newAPI(t, "company_no_jobs.json")
		jobs, _, err := source(client).FetchPage(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 0 || a.jobQueries() != 0 {
			t.Errorf("FetchPage = %d jobs, %d job queries, want none", len(jobs), a.jobQueries())
		}
	})

	t.Run("a company WTTJ no longer has is gone", func(t *testing.T) {
		t.Parallel()
		_, client := newAPI(t, "company_missing.json")
		if _, _, err := source(client).FetchPage(t.Context(), ""); !errors.Is(err, sources.ErrGone) {
			t.Errorf("FetchPage error = %v, want ErrGone", err)
		}
	})

	t.Run("a response with no data is unparseable", func(t *testing.T) {
		t.Parallel()
		a, client := newAPI(t, "company_faculty.json")
		a.company = `{"errors": [{"message": "boom"}]}`
		if _, _, err := source(client).FetchPage(t.Context(), ""); !errors.Is(err, sources.ErrUnparseable) {
			t.Errorf("FetchPage error = %v, want ErrUnparseable", err)
		}
	})

	t.Run("403 and 429 block the poll without retrying", func(t *testing.T) {
		t.Parallel()
		for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests} {
			a, client := newAPI(t, "company_faculty.json")
			a.status = status
			_, _, err := source(client).FetchPage(t.Context(), "")
			if _, blocked := sources.BlockedUntil(err, time.Now()); !blocked {
				t.Errorf("FetchPage with status %d error = %v, want a block", status, err)
			}
			if len(a.queries) != 1 {
				t.Errorf("status %d: %d requests, want 1", status, len(a.queries))
			}
		}
	})

	t.Run("a block lasts for Retry-After, or five minutes without one", func(t *testing.T) {
		t.Parallel()
		for retryAfter, want := range map[string]time.Duration{"120": 2 * time.Minute, "": 5 * time.Minute} {
			a, client := newAPI(t, "company_faculty.json")
			a.status, a.retryAfter = http.StatusTooManyRequests, retryAfter
			start := time.Now()
			_, _, err := source(client).FetchPage(t.Context(), "")
			until, blocked := sources.BlockedUntil(err, start)
			if got := until.Sub(start); !blocked || got < want-5*time.Second || got > want+5*time.Second {
				t.Errorf("Retry-After %q: blocked for %v (blocked %v), want about %v", retryAfter, got, blocked, want)
			}
		}
	})
}

func TestPollBoard(t *testing.T) {
	t.Parallel()
	t.Run("a UK company is next due in 7 days and a non-UK one in 90", func(t *testing.T) {
		t.Parallel()
		for company, want := range map[string]time.Duration{
			"company_faculty.json": 7 * 24 * time.Hour,
			"company_us_only.json": 90 * 24 * time.Hour,
		} {
			_, client := newAPI(t, company)
			res, err := source(client).PollBoard(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if res.NextPollIn != want {
				t.Errorf("PollBoard(%s).NextPollIn = %v, want %v", company, res.NextPollIn, want)
			}
		}
	})

	t.Run("the company becomes a profile", func(t *testing.T) {
		t.Parallel()
		_, client := newAPI(t, "company_faculty.json")
		res, err := source(client).PollBoard(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		want := &dto.CompanyProfile{
			Sectors:       []string{"B2B", "Artificial Intelligence", "Big data", "Machine Learning"},
			Size:          "201-500",
			Growth:        "+19%",
			FundingTotal:  "$56.1M",
			FundingRounds: 4,
			HQ:            "Old Street, London, UK",
			UKVisa:        "yes",
			Glassdoor:     "3.90",
			Mission:       "The safe, widespread adoption of AI.",
		}
		if diff := cmp.Diff(want, res.Profile); diff != "" {
			t.Errorf("PollBoard profile mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("a 429 defers every later wttj request on the same client", func(t *testing.T) {
		t.Parallel()
		a, client := newAPI(t, "company_faculty.json")
		a.status = http.StatusTooManyRequests
		if _, err := source(client).PollBoard(t.Context()); err == nil {
			t.Fatal("PollBoard on 429 = nil error, want error")
		}
		other := wttj.New("Other", func() time.Time { return now }, client)
		if _, err := other.PollBoard(t.Context()); !errors.Is(err, sources.ErrDeferred) || len(a.queries) != 1 {
			t.Errorf("PollBoard after 429 = %v with %d requests, want ErrDeferred and 1", err, len(a.queries))
		}
	})
}

func TestDiscover(t *testing.T) {
	t.Parallel()
	t.Run("a UK company returns its name, its jobs and UK", func(t *testing.T) {
		t.Parallel()
		_, client := newAPI(t, "company_faculty.json")
		got, err := source(client).Discover(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Faculty" || !got.UK || len(got.Jobs) == 0 || got.Profile == nil {
			t.Errorf("Discover() = name %q, UK %v, %d jobs, profile %v, want Faculty, UK, jobs and a profile", got.Name, got.UK, len(got.Jobs), got.Profile)
		}
	})

	t.Run("a non-UK company sends one request and is not UK", func(t *testing.T) {
		t.Parallel()
		a, client := newAPI(t, "company_us_only.json")
		got, err := source(client).Discover(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got.UK || len(got.Jobs) != 0 || len(a.queries) != 1 {
			t.Errorf("Discover() = UK %v, %d jobs, %d requests, want not UK, none, 1", got.UK, len(got.Jobs), len(a.queries))
		}
	})
}
