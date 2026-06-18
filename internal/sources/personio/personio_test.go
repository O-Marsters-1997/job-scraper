package personio

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestFetchBoard_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/jobs_acme.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var root workzagJobs
	if err := xml.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	const token = "acme"
	jobs := make([]dto.Job, 0, len(root.Jobs))
	for _, pj := range root.Jobs {
		jobs = append(jobs, dto.Job{
			Title:       pj.JobPosition,
			Location:    pj.Office,
			URL:         pj.ApplyOnline,
			CompanySlug: token,
			Source:      "personio",
			Description: pj.JobDescription,
		})
	}

	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}

	tests := []struct {
		idx         int
		title       string
		location    string
		url         string
		companySlug string
		source      string
		description string
	}{
		{
			idx:         0,
			title:       "Platform Engineer",
			location:    "London, UK",
			url:         "https://acme.personio.de/job/12345",
			companySlug: "acme",
			source:      "personio",
			description: "<p>We are looking for a Platform Engineer.</p>",
		},
		{
			idx:         1,
			title:       "Security Engineer",
			location:    "Remote",
			url:         "https://acme.personio.de/job/67890",
			companySlug: "acme",
			source:      "personio",
			description: "<p>Join our security team as a Security Engineer.</p>",
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
		if j.CompanySlug != tc.companySlug {
			t.Errorf("[%d] CompanySlug = %q, want %q", tc.idx, j.CompanySlug, tc.companySlug)
		}
		if j.Source != tc.source {
			t.Errorf("[%d] Source = %q, want %q", tc.idx, j.Source, tc.source)
		}
		if j.Description != tc.description {
			t.Errorf("[%d] Description = %q, want %q", tc.idx, j.Description, tc.description)
		}
		if j.SalaryRaw != "" {
			t.Errorf("[%d] SalaryRaw = %q, want empty", tc.idx, j.SalaryRaw)
		}
	}
}
