package cvtemplatestest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	cvtemplatestest.RunStoreContract(t, func(t *testing.T) cvtemplatestest.Fixture {
		t.Helper()
		fs := cvtemplatestest.NewFakeStore()
		tdID := fs.SeedTrackedDoc("user-1", "docA")
		return cvtemplatestest.Fixture{Store: fs, UserID: "user-1", TrackedDocID: tdID}
	})
}
