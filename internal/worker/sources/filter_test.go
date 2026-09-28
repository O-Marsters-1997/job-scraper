package sources_test

import (
	"slices"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

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
			want:     []string{},
		},
		{
			name:     "any-of multiple keywords",
			keywords: []string{"marketing", "postgres"},
			want:     []string{"Senior Backend Engineer", "Marketing Manager"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := []string{}
			for _, j := range sources.FilterByKeywords(jobs, tc.keywords) {
				got = append(got, j.Title)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
