package sourcespec_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

func TestSources(t *testing.T) {
	enumerated := map[string][]string{
		"linkedin": {"recency", "arrangement", "experience", "job_type", "distance", "salary_band"},
		"wis":      {"loc", "remote_only", "category", "contract", "hours", "salary_from"},
	}
	byName := map[string]sourcespec.SourceInfo{}
	for _, s := range sourcespec.Sources() {
		byName[s.Name] = s
	}
	for source, names := range enumerated {
		for _, name := range names {
			t.Run(source+" "+name+" has options", func(t *testing.T) {
				for _, f := range byName[source].Filters {
					if f.Name == name {
						if len(f.Options) == 0 {
							t.Errorf("%s.%s has no options", source, name)
						}
						return
					}
				}
				t.Errorf("%s.%s not declared", source, name)
			})
		}
	}
}

func TestValidFilterValue(t *testing.T) {
	tests := []struct {
		name                 string
		source, field, value string
		want                 bool
	}{
		{"declared option", "linkedin", "recency", "r86400", true},
		{"undeclared value", "linkedin", "recency", "r1", false},
		{"numeric field accepts digits", "linkedin", "company_id", "123", true},
		{"numeric field rejects text", "linkedin", "company_id", "abc", false},
		{"free-form field accepts anything", "linkedin", "location", "London", true},
		{"unknown field", "linkedin", "nope", "1", false},
		{"unknown source", "nope", "recency", "r86400", false},
		{"wis loc", "wis", "loc", "86383", true},
		{"wis removed region", "wis", "region", "uk", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sourcespec.ValidFilterValue(tt.source, tt.field, tt.value); got != tt.want {
				t.Errorf("ValidFilterValue(%q, %q, %q) = %v, want %v", tt.source, tt.field, tt.value, got, tt.want)
			}
		})
	}
}
