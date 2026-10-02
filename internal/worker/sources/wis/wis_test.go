package wis_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
)

func TestSnapshots(t *testing.T) {
	sourcetest.RunSnapshotTests(t, wis.New(wis.Search{}))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, wis.New(wis.Search{}))
}

func TestSearchURL(t *testing.T) {
	tests := []struct {
		name   string
		search wis.Search
		want   string
	}{
		{
			name:   "keywords only still sorts newest first",
			search: wis.Search{Keywords: "go"},
			want:   "https://workinstartups.com/search?per_page=50&q=go&sb=date&sd=down",
		},
		{
			name: "every filter",
			search: wis.Search{Keywords: "go", Filters: map[string]string{
				"loc": "86384", "remote_only": "1", "category": "2",
				"contract": "permanent", "hours": "full_time", "salary_from": "30000",
			}},
			want: "https://workinstartups.com/search?cat=2&cti=full_time&cty=permanent&loc=86384&per_page=50&q=go&remote_only=1&sb=date&sd=down&sf=30000",
		},
		{
			name:   "invalid value is dropped",
			search: wis.Search{Keywords: "go", Filters: map[string]string{"loc": "nowhere", "hours": "full_time"}},
			want:   "https://workinstartups.com/search?cti=full_time&per_page=50&q=go&sb=date&sd=down",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wis.StartURL(tt.search); got != tt.want {
				t.Errorf("StartURL(%+v) = %q, want %q", tt.search, got, tt.want)
			}
		})
	}
}
