package remotive

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestParse_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "feed_sample.json", "", func(body []byte, _ string) ([]dto.Job, error) {
		return parse(body)
	})
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
		want     []string
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
			got := sources.FilterByKeywords(jobs, tc.keywords)
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
