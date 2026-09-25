package workable

import (
	"os"
	"testing"
)

// jobs_pearltalent.json is a real capture from Workable's job-list API
// (POST apply.workable.com/api/v3/accounts/pearltalent/jobs), trimmed to two
// results. The list carries no URL or description; the parser builds the URL
// from token+shortcode and leaves the description empty. The second result has
// a blank city, a real edge case worth pinning.
func TestParse_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/jobs_pearltalent.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	const token = "pearltalent"
	jobs, err := parse(data, token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}

	tests := []struct {
		idx      int
		title    string
		location string
		url      string
	}{
		{
			idx:      0,
			title:    "Pearl Talent- Finance Operations Associate - I021",
			location: "Cape Town",
			url:      "https://apply.workable.com/pearltalent/j/2D4246C5C3/",
		},
		{
			idx:      1,
			title:    "Remote Credentialing Specialist for Healthcare Company",
			location: "",
			url:      "https://apply.workable.com/pearltalent/j/986DE1BC83/",
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
		if j.Source != "workable" {
			t.Errorf("[%d] Source = %q, want %q", tc.idx, j.Source, "workable")
		}
	}
}
