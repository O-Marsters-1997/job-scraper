package store_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store"
)

func TestStoreContract(t *testing.T) {
	cvtemplatestest.RunStoreContract(t, func(t *testing.T) cvtemplatestest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return cvtemplatestest.Fixture{
			Store:  store.New(pool),
			UserID: pgtest.InsertUser(t, pool),
			Other:  pgtest.InsertUser(t, pool),
		}
	})
}
