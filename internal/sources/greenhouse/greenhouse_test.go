package greenhouse

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestFetchBoard_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/board_acme.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var resp boardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	const token = "acme"
	jobs := make([]dto.Job, 0, len(resp.Jobs))
	for _, bj := range resp.Jobs {
		updatedAt := time.Now().UTC()
		if bj.UpdatedAt != "" {
			if t2, err := time.Parse(time.RFC3339, bj.UpdatedAt); err == nil {
				updatedAt = t2
			}
		}
		jobs = append(jobs, dto.Job{
			Title:       bj.Title,
			Location:    bj.Location.Name,
			URL:         bj.AbsoluteURL,
			CompanySlug: token,
			Source:      "greenhouse",
			Description: bj.Content,
			SalaryRaw:   "",
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
			title:       "Senior Product Engineer",
			location:    "London, UK",
			url:         "https://boards.greenhouse.io/acme/jobs/1001",
			companySlug: "acme",
			source:      "greenhouse",
			description: "<p>We are looking for a Senior Product Engineer...</p>",
			updatedAt:   time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			idx:         1,
			title:       "Staff Engineer",
			location:    "Remote",
			url:         "https://boards.greenhouse.io/acme/jobs/1002",
			companySlug: "acme",
			source:      "greenhouse",
			description: "<p>Join our engineering team as a Staff Engineer.</p>",
			updatedAt:   time.Date(2024, 6, 2, 10, 30, 0, 0, time.UTC),
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
