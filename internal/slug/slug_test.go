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

func TestCompany(t *testing.T) {
	tests := map[string]string{
		"Acme Ltd":             "acme",
		"Acme Limited":         "acme",
		"Acme, Inc.":           "acme",
		"Acme":                 "acme",
		"Ltd":                  "ltd",
		"Acme Holdings Co Ltd": "acme-holdings",
		"Acme - Ltd":           "acme",
		"Co Ltd":               "co",
		"Ltd Acme":             "ltd-acme",
		"Acme Corporate":       "acme-corporate",
		"":                     "",
	}
	for in, want := range tests {
		if got := slug.Company(in); got != want {
			t.Errorf("Company(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanize(t *testing.T) {
	tests := map[string]string{"acme-corp": "Acme Corp", "1password": "1password", "a--b": "A  B"}
	for in, want := range tests {
		if got := slug.Humanize(in); got != want {
			t.Errorf("Humanize(%q) = %q, want %q", in, got, want)
		}
	}
}
