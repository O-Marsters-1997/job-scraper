package remotive

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

	if len(jobs) != 3 {
		t.Fatalf("got %d jobs, want 3", len(jobs))
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
			title:       "Freelance Writer",
			location:    "Worldwide",
			url:         "https://remotive.com/remote-jobs/writing/freelance-writer-1185979",
			companySlug: "iapwe",
			salaryRaw:   "$50-$75 /hour",
			updatedAt:   time.Date(2026, 7, 4, 16, 53, 4, 0, time.UTC),
		},
		{
			idx:         1,
			title:       "Copywriter",
			location:    "Worldwide",
			url:         "https://remotive.com/remote-jobs/writing/copywriter-1749306",
			companySlug: "coalition-technologies",
			salaryRaw:   "$20k -$35k",
			updatedAt:   time.Date(2026, 7, 2, 20, 1, 13, 0, time.UTC),
		},
		{
			idx:         2,
			title:       "Senior AI Engineer",
			location:    "Northern America, LATAM, Europe, APAC",
			url:         "https://remotive.com/remote-jobs/software-development/senior-ai-engineer-2090986",
			companySlug: "lemonio",
			salaryRaw:   "",
			updatedAt:   time.Date(2026, 7, 2, 11, 45, 47, 0, time.UTC),
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
		if j.Source != "remotive" {
			t.Errorf("[%d] Source = %q, want %q", tc.idx, j.Source, "remotive")
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
