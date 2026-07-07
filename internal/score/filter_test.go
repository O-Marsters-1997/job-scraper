package score_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

func TestReject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		job          dto.Job
		cfg          dto.SearchConfig
		wantRejected bool
		wantReason   string
	}{
		{
			name:         "empty config passes",
			job:          dto.Job{Title: "Senior Software Engineer"},
			cfg:          dto.SearchConfig{},
			wantRejected: false,
		},
		{
			name:         "excluded keyword rejects whole-word match",
			job:          dto.Job{Title: "Java Engineer"},
			cfg:          dto.SearchConfig{ExcludedTitleKeywords: []string{"java"}},
			wantRejected: true,
			wantReason:   "title keyword: java",
		},
		{
			name:         "excluded keyword does not match a substring of another word",
			job:          dto.Job{Title: "JavaScript Engineer"},
			cfg:          dto.SearchConfig{ExcludedTitleKeywords: []string{"java"}},
			wantRejected: false,
		},
		{
			name:         "multi-word phrase only matches adjacent tokens",
			job:          dto.Job{Title: "Engineering Manager"},
			cfg:          dto.SearchConfig{ExcludedTitleKeywords: []string{"manager engineering"}},
			wantRejected: false,
		},
		{
			name:         "multi-word phrase matches in order",
			job:          dto.Job{Title: "Senior Engineering Manager"},
			cfg:          dto.SearchConfig{ExcludedTitleKeywords: []string{"engineering manager"}},
			wantRejected: true,
			wantReason:   "title keyword: engineering manager",
		},
		{
			name:         "c++ and c# survive tokenization as single tokens",
			job:          dto.Job{Title: "C++ Engineer"},
			cfg:          dto.SearchConfig{ExcludedTitleKeywords: []string{"c#"}},
			wantRejected: false,
		},
		{
			name:         "c++ keyword rejects c++ title",
			job:          dto.Job{Title: "C++ Engineer"},
			cfg:          dto.SearchConfig{ExcludedTitleKeywords: []string{"c++"}},
			wantRejected: true,
			wantReason:   "title keyword: c++",
		},
		{
			name:         "excluded company matches by slug",
			job:          dto.Job{Title: "Engineer", CompanySlug: "acme-corp"},
			cfg:          dto.SearchConfig{ExcludedCompanies: []string{"Acme Corp"}},
			wantRejected: true,
			wantReason:   "company: acme-corp",
		},
		{
			name:         "near-miss company passes",
			job:          dto.Job{Title: "Engineer", CompanySlug: "acme-corp"},
			cfg:          dto.SearchConfig{ExcludedCompanies: []string{"Acme Industries"}},
			wantRejected: false,
		},
		{
			name:         "excluded seniority rejects a detected level",
			job:          dto.Job{Title: "Senior Engineer"},
			cfg:          dto.SearchConfig{ExcludedSeniority: []string{"senior"}},
			wantRejected: true,
			wantReason:   "seniority: senior",
		},
		{
			name:         "ambiguous title with no seniority signal passes",
			job:          dto.Job{Title: "Software Engineer"},
			cfg:          dto.SearchConfig{ExcludedSeniority: []string{"senior"}},
			wantRejected: false,
		},
		{
			name:         "title signalling multiple levels rejects on any excluded one",
			job:          dto.Job{Title: "Senior Staff Engineer"},
			cfg:          dto.SearchConfig{ExcludedSeniority: []string{"staff"}},
			wantRejected: true,
			wantReason:   "seniority: staff",
		},
		{
			name:         "excluded location matches a term in the location string",
			job:          dto.Job{Title: "Engineer", Location: "New York, United States"},
			cfg:          dto.SearchConfig{ExcludedLocations: []string{"united states"}},
			wantRejected: true,
			wantReason:   "location: united states",
		},
		{
			name:         "empty location never matches",
			job:          dto.Job{Title: "Engineer"},
			cfg:          dto.SearchConfig{ExcludedLocations: []string{"united states"}},
			wantRejected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reason, rejected := score.Reject(tt.job, tt.cfg)
			if rejected != tt.wantRejected {
				t.Errorf("rejected = %v, want %v (reason %q)", rejected, tt.wantRejected, reason)
			}
			if tt.wantRejected && reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}
