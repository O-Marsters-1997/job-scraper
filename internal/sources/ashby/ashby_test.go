package ashby

import (
	"os"
	"testing"
	"time"
)

// board_askdragonfly.json is a real capture from the Ashby posting-api
// (api.ashbyhq.com/posting-api/job-board/askdragonfly), trimmed to two jobs
// with shortened descriptions. Keep it real-shaped: it exists to catch API
// field drift, which a hand-written fixture cannot.
func TestParse_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/board_askdragonfly.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	const token = "askdragonfly"
	jobs, err := parse(data, token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}

	tests := []struct {
		idx         int
		title       string
		location    string
		url         string
		description string
		updatedAt   time.Time
	}{
		{
			idx:         0,
			title:       "Founding Sales Development Representative",
			location:    "London",
			url:         "https://jobs.ashbyhq.com/askdragonfly/2a9fc9fb-f50b-43d3-8d3b-b86bee43e510",
			description: "<h2>This isn't a traditional SDR role</h2>",
			updatedAt:   time.Date(2026, 7, 6, 10, 44, 50, 212000000, time.UTC),
		},
		{
			idx:         1,
			title:       "Senior Product Engineer",
			location:    "Tallinn",
			url:         "https://jobs.ashbyhq.com/askdragonfly/43376c66-9659-41bf-afba-f527caa2e4f1",
			description: "<p><strong>About Us</strong></p>",
			updatedAt:   time.Date(2026, 6, 24, 14, 36, 34, 183000000, time.UTC),
		},
	}

	for _, tc := range tests {
		j := jobs[tc.idx]
		if j.Title != tc.title {
			t.Errorf("[%d] Title = %q, want %q", tc.idx, j.Title, tc.title)
		}
		if j.Location != tc.location {
			t.Errorf("[%d] Location = %q, want %q", tc.idx, j.Location, tc.location)
		}
		if j.URL != tc.url {
			t.Errorf("[%d] URL = %q, want %q", tc.idx, j.URL, tc.url)
		}
		if j.CompanySlug != token {
			t.Errorf("[%d] CompanySlug = %q, want %q", tc.idx, j.CompanySlug, token)
		}
		if j.Source != "ashby" {
			t.Errorf("[%d] Source = %q, want %q", tc.idx, j.Source, "ashby")
		}
		if j.Description != tc.description {
			t.Errorf("[%d] Description = %q, want %q", tc.idx, j.Description, tc.description)
		}
		if !j.UpdatedAt.Equal(tc.updatedAt) {
			t.Errorf("[%d] UpdatedAt = %v, want %v", tc.idx, j.UpdatedAt, tc.updatedAt)
		}
	}
}
