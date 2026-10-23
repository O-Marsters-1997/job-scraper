package cvedit_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func TestTailorViolation(t *testing.T) {
	t.Parallel()
	const original = "Built REST services in Go for the payments team"
	cases := []struct {
		name     string
		result   string
		maxChars int
		want     bool
	}{
		{"a term swap passes", "Built microservices in Go for the payments team", 100, false},
		{"punctuation and case are ignored", "built REST services in go, for the payments team.", 100, false},
		{"exactly a third of the words changed passes", "Built RPC services in Rust for the billing team", 100, false},
		{"over a third of the words changed fails", "Built RPC services in Rust for the billing group", 100, true},
		{"more words fails", "Built REST services in Go for the payments team daily", 100, true},
		{"longer than maxChars fails", "Built REST services in Go for the payments team", 20, true},
		{"a drifting rewrite fails", "Owned reliability of services for payments", 100, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cvedit.TailorViolation(original, tc.result, tc.maxChars, 1.0/3)

			if (got != "") != tc.want {
				t.Errorf("TailorViolation(%q) = %q, want violation %t", tc.result, got, tc.want)
			}
		})
	}
}
