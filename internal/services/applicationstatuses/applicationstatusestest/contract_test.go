package applicationstatusestest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses/applicationstatusestest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	applicationstatusestest.RunStoreContract(t, func(t *testing.T) applicationstatusestest.Fixture {
		t.Helper()
		return applicationstatusestest.Fixture{Store: applicationstatusestest.NewFakeStore(), UserID: "user-1"}
	})
}
