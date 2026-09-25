package slug_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/slug"
)

func TestMake(t *testing.T) {
	tests := map[string]string{
		"Acme Corp":     "acme-corp",
		"  Acme, Inc. ": "acme-inc",
		"Über-Tech Ltd": "über-tech-ltd",
		"---":           "",
		"R&D 42":        "rd-42",
	}
	for in, want := range tests {
		if got := slug.Make(in); got != want {
			t.Errorf("Make(%q) = %q, want %q", in, got, want)
		}
	}
}
