package identitytest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	identitytest.RunStoreContract(t, func(t *testing.T) identity.Store {
		t.Helper()
		return identitytest.NewFakeStore()
	})
}
