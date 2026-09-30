package filter_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
)

func TestReject(t *testing.T) {
	t.Parallel()

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			job        dto.Job
			cfg        dto.SearchConfig
			wantReason string
		}{
			{
				name:       "excluded keyword rejects whole-word match",
				job:        dto.Job{Title: "Java Engineer"},
				cfg:        dto.SearchConfig{ExcludedTitleKeywords: []string{"java"}},
				wantReason: "title keyword: java",
			},
			{
				name:       "multi-word phrase matches in order",
				job:        dto.Job{Title: "Senior Engineering Manager"},
				cfg:        dto.SearchConfig{ExcludedTitleKeywords: []string{"engineering manager"}},
				wantReason: "title keyword: engineering manager",
			},
			{
				name:       "c++ keyword rejects c++ title",
				job:        dto.Job{Title: "C++ Engineer"},
				cfg:        dto.SearchConfig{ExcludedTitleKeywords: []string{"c++"}},
				wantReason: "title keyword: c++",
			},
			{
				name:       "excluded company matches by slug",
				job:        dto.Job{Title: "Engineer", CompanySlug: "acme-corp"},
				cfg:        dto.SearchConfig{ExcludedCompanies: []string{"Acme Corp"}},
				wantReason: "company: acme-corp",
			},
			{
				name:       "excluded location matches a term in the location string",
				job:        dto.Job{Title: "Engineer", Location: "New York, United States"},
				cfg:        dto.SearchConfig{ExcludedLocations: []string{"united states"}},
				wantReason: "location: united states",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				reason, rejected := filter.Reject(tt.job, tt.cfg)
				if !rejected {
					t.Fatalf("Reject(%+v, %+v) rejected = false, want true", tt.job, tt.cfg)
				}
				if reason != tt.wantReason {
					t.Errorf("Reject(%+v, %+v) reason = %q, want %q", tt.job, tt.cfg, reason, tt.wantReason)
				}
			})
		}
	})

	t.Run("passes", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			job  dto.Job
			cfg  dto.SearchConfig
		}{
			{
				name: "empty config passes",
				job:  dto.Job{Title: "Senior Software Engineer"},
				cfg:  dto.SearchConfig{},
			},
			{
				name: "excluded keyword does not match a substring of another word",
				job:  dto.Job{Title: "JavaScript Engineer"},
				cfg:  dto.SearchConfig{ExcludedTitleKeywords: []string{"java"}},
			},
			{
				name: "multi-word phrase only matches adjacent tokens",
				job:  dto.Job{Title: "Engineering Manager"},
				cfg:  dto.SearchConfig{ExcludedTitleKeywords: []string{"manager engineering"}},
			},
			{
				name: "c++ and c# survive tokenization as single tokens",
				job:  dto.Job{Title: "C++ Engineer"},
				cfg:  dto.SearchConfig{ExcludedTitleKeywords: []string{"c#"}},
			},
			{
				name: "near-miss company passes",
				job:  dto.Job{Title: "Engineer", CompanySlug: "acme-corp"},
				cfg:  dto.SearchConfig{ExcludedCompanies: []string{"Acme Industries"}},
			},
			{
				name: "empty location never matches",
				job:  dto.Job{Title: "Engineer"},
				cfg:  dto.SearchConfig{ExcludedLocations: []string{"united states"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				if reason, rejected := filter.Reject(tt.job, tt.cfg); rejected {
					t.Errorf("Reject(%+v, %+v) = %q, true, want not rejected", tt.job, tt.cfg, reason)
				}
			})
		}
	})
}
