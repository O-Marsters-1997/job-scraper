package cvtemplatestest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	cvtemplatestest.RunStoreContract(t, func(t *testing.T) cvtemplatestest.Fixture {
		t.Helper()
		return cvtemplatestest.Fixture{Store: cvtemplatestest.NewFakeStore(), UserID: "user-1", Other: "user-2"}
	})
}
