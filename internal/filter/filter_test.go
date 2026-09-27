package filter_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
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
			name: "blocked tech rejects whole-word match in the title",
			job:  dto.Job{Title: "Kubernetes Engineer"},
			cfg: dto.SearchConfig{Preferences: dto.Preferences{
				BlockedTech: []string{"kubernetes"},
			}},
			wantRejected: true,
			wantReason:   "blocked tech: kubernetes",
		},
		{
			name: "blocked tech rejects whole-word match in the description",
			job:  dto.Job{Title: "Platform Engineer", Description: "You'll run workloads on Kubernetes daily."},
			cfg: dto.SearchConfig{Preferences: dto.Preferences{
				BlockedTech: []string{"kubernetes"},
			}},
			wantRejected: true,
			wantReason:   "blocked tech: kubernetes",
		},
		{
			name: "blocked tech does not match a substring of another word",
			job:  dto.Job{Title: "JavaScript Engineer"},
			cfg: dto.SearchConfig{Preferences: dto.Preferences{
				BlockedTech: []string{"java"},
			}},
			wantRejected: false,
		},
		{
			name: "blocked tech phrase matches adjacent tokens in order",
			job:  dto.Job{Description: "Built with Google Cloud Platform end to end."},
			cfg: dto.SearchConfig{Preferences: dto.Preferences{
				BlockedTech: []string{"google cloud platform"},
			}},
			wantRejected: true,
			wantReason:   "blocked tech: google cloud platform",
		},
		{
			name: "c++ and c# survive tokenization as single tokens",
			job:  dto.Job{Title: "C++ Engineer"},
			cfg: dto.SearchConfig{Preferences: dto.Preferences{
				BlockedTech: []string{"c#"},
			}},
			wantRejected: false,
		},
		{
			name: "c++ blocked tech rejects c++ title",
			job:  dto.Job{Title: "C++ Engineer"},
			cfg: dto.SearchConfig{Preferences: dto.Preferences{
				BlockedTech: []string{"c++"},
			}},
			wantRejected: true,
			wantReason:   "blocked tech: c++",
		},
		{
			name:         "excluded location does not affect scoring rejection",
			job:          dto.Job{Title: "Engineer", Location: "New York, United States"},
			cfg:          dto.SearchConfig{ExcludedLocations: []string{"united states"}},
			wantRejected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reason, rejected := filter.Reject(tt.job, tt.cfg)
			if rejected != tt.wantRejected {
				t.Errorf("rejected = %v, want %v (reason %q)", rejected, tt.wantRejected, reason)
			}
			if tt.wantRejected && reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestRejectLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		job          dto.Job
		cfg          dto.SearchConfig
		wantRejected bool
		wantReason   string
	}{
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
		{
			name:         "no excluded locations passes",
			job:          dto.Job{Title: "Engineer", Location: "London"},
			cfg:          dto.SearchConfig{},
			wantRejected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reason, rejected := filter.RejectLocation(tt.job, tt.cfg)
			if rejected != tt.wantRejected {
				t.Errorf("rejected = %v, want %v (reason %q)", rejected, tt.wantRejected, reason)
			}
			if tt.wantRejected && reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}
