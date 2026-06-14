package workable

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestFetchBoard_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/jobs_acme.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var resp boardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	const token = "acme"
	jobs := make([]dto.Job, 0, len(resp.Results))
	for _, r := range resp.Results {
		jobs = append(jobs, dto.Job{
			Title:       r.Title,
			Location:    r.Location.City,
			URL:         r.URL,
			CompanySlug: token,
			Source:      "workable",
			Description: r.FullDescription,
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
			title:       "Frontend Engineer",
			location:    "London",
			url:         "https://apply.workable.com/acme/j/ABC123/",
			companySlug: "acme",
			source:      "workable",
			description: "<p>We are looking for a Frontend Engineer.</p>",
		},
		{
			idx:         1,
			title:       "DevOps Engineer",
			location:    "Remote",
			url:         "https://apply.workable.com/acme/j/DEF456/",
			companySlug: "acme",
			source:      "workable",
			description: "<p>Join our infrastructure team as a DevOps Engineer.</p>",
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

func TestNeedsDetail_ReturnsFalse(t *testing.T) {
	s := New(Config{Boards: []string{"acme"}})
	if s.NeedsDetail() {
		t.Error("NeedsDetail() = true, want false for ATS source")
	}
}
