package trackeddocstest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs/trackeddocstest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	trackeddocstest.RunStoreContract(t, func(t *testing.T) trackeddocstest.Fixture {
		t.Helper()
		return trackeddocstest.Fixture{Store: trackeddocstest.NewFakeStore(), UserID: "user-1"}
	})
}
