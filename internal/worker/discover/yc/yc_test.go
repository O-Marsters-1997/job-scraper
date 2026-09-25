package yc

import (
	"os"
	"testing"
)

func TestParseCompanies(t *testing.T) {
	body, err := os.ReadFile("testdata/page1.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	companies, next, err := parseCompanies(body)
	if err != nil {
		t.Fatalf("parseCompanies: %v", err)
	}

	if len(companies) != 25 {
		t.Fatalf("got %d companies, want 25", len(companies))
	}
	if next != "https://api.ycombinator.com/v0.1/companies?page=2" {
		t.Errorf("got next %q, want page=2 URL", next)
	}

	first := companies[0]
	if first.Name != "Simantic" {
		t.Errorf("got first name %q, want Simantic", first.Name)
	}
	if first.Domain != "simantic.dev" {
		t.Errorf("got first domain %q, want simantic.dev", first.Domain)
	}
}

func TestParseCompanies_EmptyBody(t *testing.T) {
	if _, _, err := parseCompanies([]byte(`not json`)); err == nil {
		t.Error("want error for invalid JSON, got nil")
	}
}

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://www.acme.com/careers": "acme.com",
		"http://acme.com":              "acme.com",
		"acme.com":                     "acme.com",
		"":                             "",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, want %q", in, got, want)
		}
	}
}
