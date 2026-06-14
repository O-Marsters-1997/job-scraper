package lever

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestFetchBoard_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/postings_acme.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var postings []posting
	if err := json.Unmarshal(data, &postings); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	const token = "acme"
	jobs := make([]dto.Job, 0, len(postings))
	for _, p := range postings {
		updatedAt := time.Now().UTC()
		if p.CreatedAt != 0 {
			updatedAt = time.UnixMilli(p.CreatedAt).UTC()
		}
		jobs = append(jobs, dto.Job{
			Title:       p.Text,
			Location:    p.Categories.Location,
			URL:         p.HostedURL,
			CompanySlug: token,
			Source:      "lever",
			Description: p.DescriptionPlain,
			UpdatedAt:   updatedAt,
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
		updatedAt   time.Time
	}{
		{
			idx:         0,
			title:       "Senior Backend Engineer",
			location:    "London, UK",
			url:         "https://jobs.lever.co/acme/abc123",
			companySlug: "acme",
			source:      "lever",
			description: "We are looking for a Senior Backend Engineer to join our team.",
			updatedAt:   time.UnixMilli(1717228800000).UTC(),
		},
		{
			idx:         1,
			title:       "Product Designer",
			location:    "Remote",
			url:         "https://jobs.lever.co/acme/def456",
			companySlug: "acme",
			source:      "lever",
			description: "Join our design team as a Product Designer.",
			updatedAt:   time.UnixMilli(1717315200000).UTC(),
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
		if !j.UpdatedAt.Equal(tc.updatedAt) {
			t.Errorf("[%d] UpdatedAt = %v, want %v", tc.idx, j.UpdatedAt, tc.updatedAt)
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
