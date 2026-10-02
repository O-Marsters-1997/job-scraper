package wis_test

import (
	"testing"
	"time"

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

func TestRecency(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time {
		at := now.Add(-d)
		return &at
	}
	tests := []struct {
		name string
		last *time.Time
		want string
	}{
		{"first run", nil, ""},
		{"two hours ago", ago(2 * time.Hour), "1"},
		{"thirty hours ago", ago(30 * time.Hour), "3"},
		{"four days ago", ago(4 * 24 * time.Hour), "7"},
		{"nine days ago", ago(9 * 24 * time.Hour), ""},
		{"margin pushes a boundary up a tier", ago(23*time.Hour + 30*time.Minute), "3"},
		{"future last run still keeps the margin", ago(-time.Hour), "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wis.Recency(tt.last, now); got != tt.want {
				t.Errorf("Recency = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchPage_RecencyParam(t *testing.T) {
	tests := []struct {
		name    string
		recency string
		want    string
	}{
		{"sent when set", "3", "3"},
		{"omitted when empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := wis.New(wis.Search{Keywords: "go", Recency: tt.recency})
			recorder := sourcetest.Respond(`<span data-cy-count="0"></span>`)
			src.Client().Transport = recorder
			if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
				t.Fatal(err)
			}
			q := recorder.Last.URL.Query()
			if got := q.Get("f"); got != tt.want {
				t.Errorf("f = %q, want %q", got, tt.want)
			}
			if q.Has("f") != (tt.want != "") {
				t.Errorf("f present = %v, want %v", q.Has("f"), tt.want != "")
			}
		})
	}
}
