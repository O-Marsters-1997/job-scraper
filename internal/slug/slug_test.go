package slug_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/slug"
)

func TestMake(t *testing.T) {
	tests := map[string]string{
		"Acme Corp":     "acme-corp",
		"  Acme, Inc. ": "acme-inc",
		"Über-Tech Ltd": "über-tech-ltd",
		"---":           "",
		"R&D 42":        "rd-42",
		"a - b":         "a-b",
		"a--b":          "a-b",
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
	tests := map[string]string{"acme-corp": "Acme Corp", "1password": "1password", "a--b": "A  B", "élan-co": "Élan Co"}
	for in, want := range tests {
		if got := slug.Humanize(in); got != want {
			t.Errorf("Humanize(%q) = %q, want %q", in, got, want)
		}
	}
}

func FuzzMake(f *testing.F) {
	for _, seed := range []string{"Acme Corp", "a - b", "--x--", "Über", "\xff"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		once := slug.Make(in)
		if twice := slug.Make(once); twice != once {
			t.Errorf("Make(%q) = %q, but Make of that = %q", in, once, twice)
		}
		if strings.Contains(once, "--") {
			t.Errorf("Make(%q) = %q has a separator run", in, once)
		}
	})
}

func FuzzHumanize(f *testing.F) {
	for _, seed := range []string{"acme-corp", "élan", "a--b", "ǆx", "\xffa"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		if got := slug.Humanize(in); !utf8.ValidString(got) && utf8.ValidString(in) {
			t.Errorf("Humanize(%q) = %q, invalid UTF-8", in, got)
		}
	})
}
