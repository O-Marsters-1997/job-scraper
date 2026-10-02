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
