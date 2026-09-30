package cvtailortest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	cvtailortest.RunStoreContract(t, func(t *testing.T) cvtailortest.Fixture {
		t.Helper()
		return cvtailortest.Fixture{Store: cvtailortest.NewFakeStore(), UserID: "user-1", Other: "user-2", JobID: "job-1"}
	})
}
