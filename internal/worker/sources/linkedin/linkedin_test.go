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
	sourcetest.RunSnapshotTests(t, linkedin.New(linkedin.Search{}))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, linkedin.New(linkedin.Search{}))
}

func TestFetchPage_SearchQuery(t *testing.T) {
	tests := []struct {
		name   string
		search linkedin.Search
		want   map[string]string
	}{
		{
			name:   "keywords only, no filters set",
			search: linkedin.Search{Keywords: "engineer"},
			want:   map[string]string{"keywords": "engineer"},
		},
		{
			name:   "location set",
			search: linkedin.Search{Keywords: "engineer", Location: "London"},
			want:   map[string]string{"keywords": "engineer", "location": "London"},
		},
		{
			name:   "valid company_id",
			search: linkedin.Search{CompanyID: "12345"},
			want:   map[string]string{"f_C": "12345"},
		},
		{
			name:   "invalid company_id dropped",
			search: linkedin.Search{CompanyID: "not-a-number"},
			want:   map[string]string{},
		},
		{
			name:   "valid recency",
			search: linkedin.Search{Recency: "r86400"},
			want:   map[string]string{"f_TPR": "r86400"},
		},
		{
			name:   "invalid recency dropped",
			search: linkedin.Search{Recency: "r1"},
			want:   map[string]string{},
		},
		{
			name:   "valid arrangement",
			search: linkedin.Search{Arrangement: "2"},
			want:   map[string]string{"f_WT": "2"},
		},
		{
			name:   "invalid arrangement dropped",
			search: linkedin.Search{Arrangement: "9"},
			want:   map[string]string{},
		},
		{
			name:   "valid experience",
			search: linkedin.Search{Experience: "3"},
			want:   map[string]string{"f_E": "3"},
		},
		{
			name:   "invalid experience dropped",
			search: linkedin.Search{Experience: "7"},
			want:   map[string]string{},
		},
		{
			name:   "valid job_type",
			search: linkedin.Search{JobType: "F"},
			want:   map[string]string{"f_JT": "F"},
		},
		{
			name:   "invalid job_type dropped",
			search: linkedin.Search{JobType: "X"},
			want:   map[string]string{},
		},
		{
			name:   "valid geo_id",
			search: linkedin.Search{GeoID: "103644278"},
			want:   map[string]string{"geoId": "103644278"},
		},
		{
			name:   "invalid geo_id dropped",
			search: linkedin.Search{GeoID: "abc"},
			want:   map[string]string{},
		},
		{
			name:   "valid distance",
			search: linkedin.Search{Distance: "25"},
			want:   map[string]string{"f_D": "25"},
		},
		{
			name:   "invalid distance dropped",
			search: linkedin.Search{Distance: "far"},
			want:   map[string]string{},
		},
		{
			name:   "valid salary_band",
			search: linkedin.Search{SalaryBand: "5"},
			want:   map[string]string{"f_SB2": "5"},
		},
		{
			name:   "invalid salary_band dropped",
			search: linkedin.Search{SalaryBand: "0"},
			want:   map[string]string{},
		},
		{
			name: "all filters set together",
			search: linkedin.Search{
				Keywords:    "engineer",
				Location:    "London",
				CompanyID:   "12345",
				Recency:     "r86400",
				Arrangement: "2",
				Experience:  "3",
				JobType:     "F",
				GeoID:       "103644278",
				Distance:    "25",
				SalaryBand:  "5",
			},
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
			src := linkedin.New(tt.search)
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
