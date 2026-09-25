package linkedin

import (
	"net/url"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func TestSnapshots(t *testing.T) {
	sources.RunSnapshotTests(t, New(Config{}))
}

func TestPageURL(t *testing.T) {
	tests := []struct {
		name   string
		search Search
		want   map[string]string
	}{
		{
			name:   "keywords only, no filters set",
			search: Search{Keywords: "engineer"},
			want:   map[string]string{"keywords": "engineer"},
		},
		{
			name:   "location set",
			search: Search{Keywords: "engineer", Location: "London"},
			want:   map[string]string{"keywords": "engineer", "location": "London"},
		},
		{
			name:   "valid company_id",
			search: Search{CompanyID: "12345"},
			want:   map[string]string{"f_C": "12345"},
		},
		{
			name:   "invalid company_id dropped",
			search: Search{CompanyID: "not-a-number"},
			want:   map[string]string{},
		},
		{
			name:   "valid recency",
			search: Search{Recency: "r86400"},
			want:   map[string]string{"f_TPR": "r86400"},
		},
		{
			name:   "invalid recency dropped",
			search: Search{Recency: "r1"},
			want:   map[string]string{},
		},
		{
			name:   "valid arrangement",
			search: Search{Arrangement: "2"},
			want:   map[string]string{"f_WT": "2"},
		},
		{
			name:   "invalid arrangement dropped",
			search: Search{Arrangement: "9"},
			want:   map[string]string{},
		},
		{
			name:   "valid experience",
			search: Search{Experience: "3"},
			want:   map[string]string{"f_E": "3"},
		},
		{
			name:   "invalid experience dropped",
			search: Search{Experience: "7"},
			want:   map[string]string{},
		},
		{
			name:   "valid job_type",
			search: Search{JobType: "F"},
			want:   map[string]string{"f_JT": "F"},
		},
		{
			name:   "invalid job_type dropped",
			search: Search{JobType: "X"},
			want:   map[string]string{},
		},
		{
			name:   "valid geo_id",
			search: Search{GeoID: "103644278"},
			want:   map[string]string{"geoId": "103644278"},
		},
		{
			name:   "invalid geo_id dropped",
			search: Search{GeoID: "abc"},
			want:   map[string]string{},
		},
		{
			name:   "valid distance",
			search: Search{Distance: "25"},
			want:   map[string]string{"f_D": "25"},
		},
		{
			name:   "invalid distance dropped",
			search: Search{Distance: "far"},
			want:   map[string]string{},
		},
		{
			name:   "valid salary_band",
			search: Search{SalaryBand: "5"},
			want:   map[string]string{"f_SB2": "5"},
		},
		{
			name:   "invalid salary_band dropped",
			search: Search{SalaryBand: "0"},
			want:   map[string]string{},
		},
		{
			name: "all filters set together",
			search: Search{
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
			raw := tt.search.pageURL(0)
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("pageURL produced invalid URL: %v", err)
			}
			query := u.Query()

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
