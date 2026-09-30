package linkedin_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSnapshots(t *testing.T) {
	sourcetest.RunSnapshotTests(t, linkedin.New("", nil))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, linkedin.New("", nil))
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@brd.superproxy.io:33335")
			t.Setenv("BRIGHTDATA_CA_CERT", "")
			var requested *url.URL
			src := linkedin.New(tt.keywords, tt.filters)
			src.Client().Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requested = r.URL
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("<html></html>"))}, nil
			})
			if _, _, err := src.FetchPage(context.Background(), ""); err != nil {
				t.Fatal(err)
			}
			query := requested.Query()

			for param, want := range tt.want {
				got := query.Get(param)
				if got != want {
					t.Errorf("param %q = %q, want %q", param, got, want)
				}
			}

			allParams := []string{"f_C", "f_TPR", "f_WT", "f_E", "f_JT", "geoId", "f_D", "f_SB2"}
			for _, param := range allParams {
				if _, wanted := tt.want[param]; wanted {
					continue
				}
				if got := query.Get(param); got != "" {
					t.Errorf("param %q = %q, want unset", param, got)
				}
			}
		})
	}
}
