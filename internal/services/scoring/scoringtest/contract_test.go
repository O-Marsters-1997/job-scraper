package scoringtest_test

import (
	"fmt"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	scoringtest.RunStoreContract(t, func(t *testing.T) scoringtest.Fixture {
		t.Helper()
		var n int
		return scoringtest.Fixture{
			Store:   scoringtest.NewFakeStore(),
			NewUser: func() string { n++; return fmt.Sprintf("user-%d", n) },
		}
	})
}
