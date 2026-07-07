package remoteok

import (
	"os"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestParse_ParsesFixture(t *testing.T) {
	data, err := os.ReadFile("snapshots/feed_sample.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	jobs, err := parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(jobs) != 4 {
		t.Fatalf("got %d jobs, want 4 (legal notice element must be skipped)", len(jobs))
	}

	tests := []struct {
		idx         int
		title       string
		location    string
		url         string
		companySlug string
		salaryRaw   string
		updatedAt   time.Time
	}{
		{
			idx:         0,
			title:       "Human Resources Generalist Manager",
			location:    "Nashville, Nashville, Tennessee, United States",
			url:         "https://remoteOK.com/remote-jobs/remote-human-resources-generalist-manager-seasoned-recruitment-1134533",
			companySlug: "seasoned-recruitment",
			salaryRaw:   "",
			updatedAt:   time.Date(2026, 7, 5, 20, 53, 39, 0, time.UTC),
		},
		{
			idx:         1,
			title:       "Content Creator",
			location:    "England, England, United Kingdom",
			url:         "https://remoteOK.com/remote-jobs/remote-content-creator-medicomedics-1134523",
			companySlug: "medicomedics",
			salaryRaw:   "",
			updatedAt:   time.Date(2026, 7, 5, 19, 42, 16, 0, time.UTC),
		},
		{
			idx:         2,
			title:       "Business Analyst",
			location:    "New York, New York, United States",
			url:         "https://remoteOK.com/remote-jobs/remote-business-analyst-rotaract-club-of-nibm-kandy-1134504",
			companySlug: "rotaract-club-of-nibm-kandy",
			salaryRaw:   "",
			updatedAt:   time.Date(2026, 7, 5, 17, 18, 41, 0, time.UTC),
		},
		{
			idx:         3,
			title:       "Senior Quality Engineer",
			location:    "",
			url:         "https://remoteOK.com/remote-jobs/remote-senior-quality-engineer-tellent-1134436",
			companySlug: "tellent",
			salaryRaw:   "$200000 - $225000",
			updatedAt:   time.Date(2026, 7, 3, 16, 0, 13, 0, time.UTC),
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
		if j.Source != "remoteok" {
			t.Errorf("[%d] Source = %q, want %q", tc.idx, j.Source, "remoteok")
		}
		if j.WorkArrangement != "remote" {
			t.Errorf("[%d] WorkArrangement = %q, want %q", tc.idx, j.WorkArrangement, "remote")
		}
		if j.SalaryRaw != tc.salaryRaw {
			t.Errorf("[%d] SalaryRaw = %q, want %q", tc.idx, j.SalaryRaw, tc.salaryRaw)
		}
		if !j.UpdatedAt.Equal(tc.updatedAt) {
			t.Errorf("[%d] UpdatedAt = %v, want %v", tc.idx, j.UpdatedAt, tc.updatedAt)
		}
		if j.Description == "" {
			t.Errorf("[%d] Description is empty, want non-empty", tc.idx)
		}
	}
}

func TestFilterByKeywords(t *testing.T) {
	jobs := []dto.Job{
		{Title: "Senior Backend Engineer", Description: "Go and Postgres"},
		{Title: "Marketing Manager", Description: "SEO and content"},
		{Title: "Support Rep", Description: "loves engineering culture"},
	}

	tests := []struct {
		name     string
		keywords []string
		want     []string // titles expected to pass
	}{
		{
			name:     "no keywords passes everything",
			keywords: nil,
			want:     []string{"Senior Backend Engineer", "Marketing Manager", "Support Rep"},
		},
		{
			name:     "matches title case-insensitively",
			keywords: []string{"engineer"},
			want:     []string{"Senior Backend Engineer", "Support Rep"},
		},
		{
			name:     "matches description",
			keywords: []string{"postgres"},
			want:     []string{"Senior Backend Engineer"},
		},
		{
			name:     "no match yields empty",
			keywords: []string{"blockchain"},
			want:     nil,
		},
		{
			name:     "any-of multiple keywords",
			keywords: []string{"marketing", "postgres"},
			want:     []string{"Senior Backend Engineer", "Marketing Manager"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterByKeywords(jobs, tc.keywords)
			gotTitles := make([]string, 0, len(got))
			for _, j := range got {
				gotTitles = append(gotTitles, j.Title)
			}
			if len(gotTitles) != len(tc.want) {
				t.Fatalf("got %v, want %v", gotTitles, tc.want)
			}
			for i, title := range gotTitles {
				if title != tc.want[i] {
					t.Errorf("got %v, want %v", gotTitles, tc.want)
				}
			}
		})
	}
}
