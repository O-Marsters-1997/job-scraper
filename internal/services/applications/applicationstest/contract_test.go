package applicationstest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	applicationstest.RunStoreContract(t, func(t *testing.T) applicationstest.Fixture {
		t.Helper()
		return applicationstest.Fixture{Store: applicationstest.NewFakeStore(), UserID: "user-1", JobID: "job-1"}
	})
}
