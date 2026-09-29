package tailoringtest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/tailoring/tailoringtest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	tailoringtest.RunStoreContract(t, func(t *testing.T) tailoringtest.Fixture {
		t.Helper()
		return tailoringtest.Fixture{Store: tailoringtest.NewFakeStore(), UserID: "user-1", Other: "user-2"}
	})
}
