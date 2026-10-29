package checks_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

func TestClosestSentence(t *testing.T) {
	t.Parallel()
	ad := "We are a friendly team. You will build Kubernetes platform tooling. You will own the on-call rota."
	for _, tc := range []struct {
		name, bullet, want string
		ok                 bool
	}{
		{"best sentence", "Built Kubernetes tooling for the platform", "You will build Kubernetes platform tooling", true},
		{"stopwords ignored", "You will be a part of the", "", false},
		{"below threshold", "Wrote Kubernetes docs", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := checks.ClosestSentence(tc.bullet, ad)
			if ok != tc.ok || got != tc.want {
				t.Errorf("ClosestSentence() = %q, %t, want %q, %t", got, ok, tc.want, tc.ok)
			}
		})
	}
}
