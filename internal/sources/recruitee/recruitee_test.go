package recruitee

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestFetchBoard_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/offers_acme.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var resp boardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	const token = "acme"
	jobs := make([]dto.Job, 0, len(resp.Offers))
	for _, o := range resp.Offers {
		jobs = append(jobs, dto.Job{
			Title:       o.Title,
			Location:    o.Location,
			URL:         o.CareersURL,
			CompanySlug: token,
			Source:      "recruitee",
			Description: o.Description,
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
			title:       "Go Engineer",
			location:    "London, UK",
			url:         "https://acme.recruitee.com/o/go-engineer",
			companySlug: "acme",
			source:      "recruitee",
			description: "<p>We are looking for a Go Engineer.</p>",
		},
		{
			idx:         1,
			title:       "ML Engineer",
			location:    "Remote",
			url:         "https://acme.recruitee.com/o/ml-engineer",
			companySlug: "acme",
			source:      "recruitee",
			description: "<p>Join our ML team as an ML Engineer.</p>",
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
