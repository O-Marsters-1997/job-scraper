package linkedin_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestSnapshots(t *testing.T) {
	sourcetest.RunSnapshotTests(t, linkedin.New("", nil, ""))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, linkedin.New("", nil, ""))
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
			name:     "valid arrangement",
			keywords: "", filters: map[string]string{"arrangement": "2"},
			want: map[string]string{"f_WT": "2"},
		},
		{
			name:     "invalid arrangement dropped",
			keywords: "", filters: map[string]string{"arrangement": "9"},
			want: map[string]string{},
		},
		{
			name:     "valid experience",
			keywords: "", filters: map[string]string{"experience": "3"},
			want: map[string]string{"f_E": "3"},
		},
		{
			name:     "invalid experience dropped",
			keywords: "", filters: map[string]string{"experience": "7"},
			want: map[string]string{},
		},
		{
			name:     "valid job_type",
			keywords: "", filters: map[string]string{"job_type": "F"},
			want: map[string]string{"f_JT": "F"},
		},
		{
			name:     "invalid job_type dropped",
			keywords: "", filters: map[string]string{"job_type": "X"},
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
			name:     "valid salary_band",
			keywords: "", filters: map[string]string{"salary_band": "5"},
			want: map[string]string{"f_SB2": "5"},
		},
		{
			name:     "invalid salary_band dropped",
			keywords: "", filters: map[string]string{"salary_band": "0"},
			want: map[string]string{},
		},
		{
			name:     "all filters set together",
			keywords: "engineer", filters: map[string]string{"location": "London", "company_id": "12345", "recency": "r86400", "arrangement": "2", "experience": "3", "job_type": "F", "geo_id": "103644278", "distance": "25", "salary_band": "5"},
			want: map[string]string{
				"keywords": "engineer",
				"location": "London",
				"f_C":      "12345",
				"f_TPR":    "r86400",
				"f_WT":     "2",
				"f_E":      "3",
				"f_JT":     "F",
				"geoId":    "103644278",
				"f_D":      "25",
				"f_SB2":    "5",
			},
		},
	}

	t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "")
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	allParams := []string{"keywords", "location", "f_C", "f_TPR", "f_WT", "f_E", "f_JT", "geoId", "f_D", "f_SB2"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := linkedin.New(tt.keywords, tt.filters, "")
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

func TestGetDetails_FetchesGuestFragment(t *testing.T) {
	html, err := os.ReadFile("snapshots/detail_details.html")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "")
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	src := linkedin.New("", nil, "")
	recorder := sourcetest.Respond(string(html))
	src.Client().Transport = recorder

	job, err := src.GetDetails(t.Context(), "https://www.linkedin.com/jobs/view/4335312853")
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
	t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "")
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	filters := map[string]string{"recency": "r604800"}
	src := linkedin.New("go", filters, "r176400")
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
