package personio

import (
	"os"
	"testing"
)

// jobs_optiply.xml mirrors the current Personio feed shape
// ({token}.jobs.personio.com/xml): <workzag-jobs> of <position> elements with
// <name>, <office>, and a nested <jobDescriptions>. There is no apply-URL
// element, so the parser builds the URL from token+id and concatenates the
// description blocks.
func TestParse_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/jobs_optiply.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	const token = "optiply"
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
	}{
		{
			idx:         0,
			title:       "Account Executive",
			location:    "Amsterdam",
			url:         "https://optiply.jobs.personio.com/job/1953132",
			description: "<p>Optiply is the AI-native platform.</p><p>We are hiring an Account Executive.</p>",
		},
		{
			idx:         1,
			title:       "Commercial Lead -BeNeLux",
			location:    "Amsterdam",
			url:         "https://optiply.jobs.personio.com/job/2623060",
			description: "<div>Join the commercial team.</div>",
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
		if j.Description != tc.description {
			t.Errorf("[%d] Description = %q, want %q", tc.idx, j.Description, tc.description)
		}
		if j.CompanySlug != token {
			t.Errorf("[%d] CompanySlug = %q, want %q", tc.idx, j.CompanySlug, token)
		}
		if j.Source != "personio" {
			t.Errorf("[%d] Source = %q, want %q", tc.idx, j.Source, "personio")
		}
	}
}
