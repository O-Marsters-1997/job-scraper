package eventstest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/events/eventstest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	eventstest.RunStoreContract(t, func(t *testing.T) eventstest.Fixture {
		t.Helper()
		return eventstest.Fixture{
			Store:       eventstest.NewFakeStore(),
			UserID:      "user-1",
			OtherUserID: "user-2",
			SubjectID:   "job-1",
		}
	})
}
