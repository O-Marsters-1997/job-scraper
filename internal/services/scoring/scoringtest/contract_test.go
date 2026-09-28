package scoringtest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	scoringtest.RunStoreContract(t, func(t *testing.T) scoring.Store {
		t.Helper()
		return scoringtest.NewFakeStore()
	})
}
