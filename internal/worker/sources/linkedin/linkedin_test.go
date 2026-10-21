package linkedin_test

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func noGap() *sources.Gate { return sources.NewGate(0, 0) }

func TestParseURLsAt(t *testing.T) {
	scrapedAt := time.Date(2026, 10, 13, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		name     string
		datetime string
		text     string
		want     time.Time
	}{
		{"hours ago keeps hour precision", "2026-10-13", "3 hours ago", scrapedAt.Add(-3 * time.Hour)},
		{"minutes ago on the same day", "2026-10-13", "20 minutes ago", scrapedAt.Add(-20 * time.Minute)},
		{"estimate before the datetime day clamps to its start", "2026-10-13", "20 hours ago", time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)},
		{"estimate after the datetime day clamps to its end", "2026-10-04", "1 week ago", time.Date(2026, 10, 4, 23, 59, 59, 0, time.UTC)},
		{"days ago inside the datetime day", "2026-10-11", "2 days ago", time.Date(2026, 10, 11, 15, 30, 0, 0, time.UTC)},
		{"unparseable text falls back to the datetime day", "2026-10-09", "Be an early applicant", time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<div class="base-search-card" data-entity-urn="urn:li:jobPosting:1"><h3 class="base-search-card__title">Engineer</h3>` +
				`<time class="job-search-card__listdate" datetime="` + tt.datetime + `">` + tt.text + `</time></div>`
			jobs, err := linkedin.ParseURLsAt(strings.NewReader(html), scrapedAt)
			if err != nil || len(jobs) != 1 {
				t.Fatalf("ParseURLsAt(...) = %v, %v, want one card", jobs, err)
			}
			if got := jobs[0].PostedAt; got == nil || !got.Equal(tt.want) {
				t.Errorf("PostedAt = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseURLs_CountryOnlyLocationIsRemote(t *testing.T) {
	tests := []struct {
		location string
		want     string
	}{
		{"United Kingdom", "remote"},
		{"London Area, United Kingdom", ""},
		{"Northern Ireland, United Kingdom", ""},
		{"Glasgow City, Scotland, United Kingdom", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.location, func(t *testing.T) {
			html := `<div class="base-search-card" data-entity-urn="urn:li:jobPosting:1"><h3 class="base-search-card__title">Engineer</h3>` +
				`<span class="job-search-card__location">` + tt.location + `</span></div>`
			jobs, err := linkedin.ParseURLs(strings.NewReader(html))
			if err != nil || len(jobs) != 1 {
				t.Fatalf("ParseURLs(...) = %v, %v, want one card", jobs, err)
			}
			if got := jobs[0].WorkArrangement; got != tt.want {
				t.Errorf("WorkArrangement for %q = %q, want %q", tt.location, got, tt.want)
			}
		})
	}
}

func TestSnapshots(t *testing.T) {
	sourcetest.RunSnapshotTests(t, linkedin.New("", nil, "", noGap()))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, linkedin.New("", nil, "", noGap()))
}

func TestFetchPage_SearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		keywords string
		filters  map[string]string
		want     map[string]string
	}{
		{
			name:     "keywords only, no filters set",
			keywords: "engineer",
			want:     map[string]string{"keywords": "engineer"},
		},
		{
			name:     "location set",
			keywords: "engineer", filters: map[string]string{"location": "London"},
			want: map[string]string{"keywords": "engineer", "location": "London"},
		},
		{
			name:     "valid company_id",
			keywords: "", filters: map[string]string{"company_id": "12345"},
			want: map[string]string{"f_C": "12345"},
		},
		{
			name:     "invalid company_id dropped",
			keywords: "", filters: map[string]string{"company_id": "not-a-number"},
			want: map[string]string{},
		},
		{
			name:     "valid recency",
			keywords: "", filters: map[string]string{"recency": "r86400"},
			want: map[string]string{"f_TPR": "r86400"},
		},
		{
			name:     "invalid recency dropped",
			keywords: "", filters: map[string]string{"recency": "r1"},
			want: map[string]string{},
		},
		{
			name:     "arrangement and city_cards are enforced locally, not sent",
			keywords: "", filters: map[string]string{"arrangement": "2", "city_cards": "full_page"},
			want: map[string]string{},
		},
		{
			name:     "valid geo_id",
			keywords: "", filters: map[string]string{"geo_id": "103644278"},
			want: map[string]string{"geoId": "103644278"},
		},
		{
			name:     "invalid geo_id dropped",
			keywords: "", filters: map[string]string{"geo_id": "abc"},
			want: map[string]string{},
		},
		{
			name:     "valid distance",
			keywords: "", filters: map[string]string{"distance": "25"},
			want: map[string]string{"f_D": "25"},
		},
		{
			name:     "invalid distance dropped",
			keywords: "", filters: map[string]string{"distance": "far"},
			want: map[string]string{},
		},
		{
			name:     "all filters set together",
			keywords: "engineer", filters: map[string]string{"location": "London", "company_id": "12345", "recency": "r86400", "arrangement": "2", "geo_id": "103644278", "distance": "25"},
			want: map[string]string{
				"keywords": "engineer",
				"location": "London",
				"f_C":      "12345",
				"f_TPR":    "r86400",
				"geoId":    "103644278",
				"f_D":      "25",
			},
		},
	}

	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	allParams := []string{"keywords", "location", "f_C", "f_TPR", "f_WT", "f_E", "f_JT", "geoId", "f_D", "f_SB2"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := linkedin.New(tt.keywords, tt.filters, "", noGap())
			recorder := sourcetest.Respond("<html></html>")
			src.Client().Transport = recorder
			if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
				t.Fatal(err)
			}
			query := recorder.Last.URL.Query()
			got := map[string]string{}
			for _, param := range allParams {
				if v := query.Get(param); v != "" {
					got[param] = v
				}
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("query params (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseJobDetailAt(t *testing.T) {
	t.Parallel()
	html, err := os.ReadFile("snapshots/detail_details.html")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	job, err := linkedin.ParseJobDetailAt(bytes.NewReader(html), "https://www.linkedin.com/jobs/view/1", now)
	if err != nil {
		t.Fatal(err)
	}

	want := now.Add(-10 * 30 * 24 * time.Hour)
	if job.PostedAt == nil || !job.PostedAt.Equal(want) {
		t.Errorf("PostedAt = %v, want %v", job.PostedAt, want)
	}
}

func TestGetDetails_FetchesGuestFragment(t *testing.T) {
	html, err := os.ReadFile("snapshots/detail_details.html")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	src := linkedin.New("", nil, "", noGap())
	recorder := sourcetest.Respond(string(html))
	src.Client().Transport = recorder

	job, err := src.GetDetails(t.Context(), dto.Job{URL: "https://www.linkedin.com/jobs/view/4335312853"})
	if err != nil {
		t.Fatal(err)
	}

	wantFetched := "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting/4335312853"
	if got := recorder.Last.URL.String(); got != wantFetched {
		t.Errorf("fetched URL = %q, want %q", got, wantFetched)
	}
	wantURL := "https://www.linkedin.com/jobs/view/4335312853"
	if job.URL != wantURL {
		t.Errorf("Job.URL = %q, want %q", job.URL, wantURL)
	}
}

func TestGetDetails_RemoteCheck(t *testing.T) {
	const (
		slugPage = "https://uk.linkedin.com/jobs/view/senior-software-developer-stream-processing-at-ibm-4461428214"
		cardURL  = "https://www.linkedin.com/jobs/view/4461428214"
	)
	fetch := func(t *testing.T, snapshot string, card dto.Job) (dto.Job, *sourcetest.Responder, error) {
		t.Helper()
		html, err := os.ReadFile("snapshots/" + snapshot)
		if err != nil {
			t.Fatal(err)
		}
		src := linkedin.New("", nil, "", noGap())
		recorder := sourcetest.Respond(string(html))
		src.Client().Transport = recorder
		job, err := src.GetDetails(t.Context(), card)
		return job, recorder, err
	}

	t.Run("a TELECOMMUTE page is kept as remote and fetched in place of the fragment", func(t *testing.T) {
		t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
		job, recorder, err := fetch(t, "detail_telecommute.html", dto.Job{URL: cardURL, RemoteCheckURL: slugPage})
		if err != nil {
			t.Fatal(err)
		}
		if got := recorder.Last.URL.String(); got != slugPage {
			t.Errorf("fetched %q, want %q", got, slugPage)
		}
		if job.WorkArrangement != "remote" || job.URL != cardURL || job.Description == "" {
			t.Errorf("job = {WorkArrangement: %q, URL: %q, Description empty: %t}, want remote, %q, a description", job.WorkArrangement, job.URL, job.Description == "", cardURL)
		}
	})

	t.Run("a page without TELECOMMUTE is not remote", func(t *testing.T) {
		t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
		_, _, err := fetch(t, "detail_not_telecommute.html", dto.Job{URL: cardURL, RemoteCheckURL: slugPage})
		if !errors.Is(err, sources.ErrNotRemote) {
			t.Errorf("GetDetails() err = %v, want ErrNotRemote", err)
		}
	})

	t.Run("a country-only card stays remote on the fragment", func(t *testing.T) {
		t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
		job, recorder, err := fetch(t, "detail_not_telecommute.html", dto.Job{URL: cardURL, WorkArrangement: "remote"})
		if err != nil {
			t.Fatal(err)
		}
		if want := "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting/4461428214"; recorder.Last.URL.String() != want {
			t.Errorf("fetched %q, want %q", recorder.Last.URL, want)
		}
		if job.WorkArrangement != "remote" {
			t.Errorf("WorkArrangement = %q, want remote", job.WorkArrangement)
		}
	})
}

func TestGetDetails_RemoteCheckJSONLD(t *testing.T) {
	const (
		slugPage = "https://uk.linkedin.com/jobs/view/engineer-at-acme-1"
		cardURL  = "https://www.linkedin.com/jobs/view/1"
		title    = `<h1 class="top-card-layout__title topcard__title">Engineer</h1>`
	)
	ld := func(body string) string {
		return `<script type="application/ld+json">` + body + `</script>`
	}
	tests := []struct {
		name       string
		page       string
		wantRemote bool
	}{
		{"no JSON-LD block", title, false},
		{"JobPosting without jobLocationType", title + ld(`{"@type":"JobPosting"}`), false},
		{"TELECOMMUTE in the second block only", title + ld(`{"@type":"Organization"}`) + ld(`{"@type":"JobPosting","jobLocationType":"TELECOMMUTE"}`), true},
		{"TELECOMMUTE on a block that is not a JobPosting", title + ld(`{"@type":"Organization","jobLocationType":"TELECOMMUTE"}`), false},
		{"malformed block is skipped", title + ld(`{not json`) + ld(`{"@type":"JobPosting","jobLocationType":"TELECOMMUTE"}`), true},
	}
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := linkedin.New("", nil, "", noGap())
			src.Client().Transport = sourcetest.Respond(tt.page)
			job, err := src.GetDetails(t.Context(), dto.Job{URL: cardURL, RemoteCheckURL: slugPage})
			if tt.wantRemote {
				if err != nil || job.WorkArrangement != "remote" {
					t.Errorf("GetDetails() = %q, %v, want remote", job.WorkArrangement, err)
				}
				return
			}
			if !errors.Is(err, sources.ErrNotRemote) {
				t.Errorf("GetDetails() err = %v, want ErrNotRemote", err)
			}
		})
	}
}

type stampedResponder struct {
	mu    sync.Mutex
	times []time.Time
}

func (r *stampedResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.times = append(r.times, time.Now())
	r.mu.Unlock()
	return sourcetest.Respond("<html></html>").RoundTrip(req)
}

func TestListingAndDetailFetchesShareOneGap(t *testing.T) {
	const gap = 80 * time.Millisecond
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	shared := sources.NewGate(gap, 0)
	stamps := &stampedResponder{}
	listing := linkedin.New("go", nil, "", shared)
	listing.Client().Transport = stamps
	detail := linkedin.New("", nil, "", shared)
	detail.Client().Transport = stamps

	if _, _, err := listing.FetchPage(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"1", "2"} {
		_, _ = detail.GetDetails(t.Context(), dto.Job{URL: "https://www.linkedin.com/jobs/view/" + id})
	}

	stamps.mu.Lock()
	defer stamps.mu.Unlock()
	if len(stamps.times) != 3 {
		t.Fatalf("requests = %d, want 3", len(stamps.times))
	}
	for i := 1; i < len(stamps.times); i++ {
		if got := stamps.times[i].Sub(stamps.times[i-1]); got < gap/2 {
			t.Errorf("gap before request %d = %v, want at least %v", i+1, got, gap/2)
		}
	}
}

type blockedTransport struct{ calls int }

func (b *blockedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	b.calls++
	return nil, proxy.ErrBlocked
}

func TestResidentialBlockPausesEveryTask(t *testing.T) {
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	shared := sources.NewGate(0, time.Minute)
	blocked := &blockedTransport{}
	first := linkedin.New("go", nil, "", shared)
	first.Client().Transport = blocked
	second := linkedin.New("", nil, "", shared)
	second.Client().Transport = blocked

	if _, _, err := first.FetchPage(t.Context(), ""); !errors.Is(err, proxy.ErrBlocked) {
		t.Fatalf("FetchPage() error = %v, want ErrBlocked", err)
	}
	_, err := second.GetDetails(t.Context(), dto.Job{URL: "https://www.linkedin.com/jobs/view/1"})

	if !errors.Is(err, sources.ErrDeferred) {
		t.Errorf("GetDetails() error = %v, want ErrDeferred", err)
	}
	if blocked.calls != 1 {
		t.Errorf("requests sent = %d, want 1", blocked.calls)
	}
}

func TestClientTimeoutCoversBlockedRetries(t *testing.T) {
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	if got := linkedin.New("", nil, "", noGap()).Client().Timeout; got < 45*time.Second {
		t.Errorf("client timeout = %v, want at least 45s to fit three attempts and their backoff", got)
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
		{"no previous success keeps stored recency", "r604800", nil, ""},
		{"two days ago adds the margin", "r604800", ago(48 * time.Hour), "r176400"},
		{"gap longer than the Target recency keeps stored recency", "r86400", ago(72 * time.Hour), ""},
		{"no configured recency still narrows", "", ago(48 * time.Hour), "r176400"},
		{"just ran still keeps the margin", "r604800", ago(0), "r3600"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := linkedin.Recency(tt.configured, tt.last, now); got != tt.want {
				t.Errorf("Recency(%q) = %q, want %q", tt.configured, got, tt.want)
			}
		})
	}
}

func TestFetchPage_NarrowedRecencyOverridesFilter(t *testing.T) {
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	filters := map[string]string{"recency": "r604800"}
	src := linkedin.New("go", filters, "r176400", noGap())
	recorder := sourcetest.Respond("<html></html>")
	src.Client().Transport = recorder
	if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if got := recorder.Last.URL.Query().Get("f_TPR"); got != "r176400" {
		t.Errorf("f_TPR = %q, want r176400", got)
	}
	if filters["recency"] != "r604800" {
		t.Errorf("stored recency = %q, want r604800", filters["recency"])
	}
}
