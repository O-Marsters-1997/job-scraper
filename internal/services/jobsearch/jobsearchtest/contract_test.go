package jobsearchtest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	jobsearchtest.RunStoreContract(t, func(t *testing.T) (jobsearchtest.Store, string) {
		t.Helper()
		return jobsearchtest.NewFakeStore(), "contract-user"
	})
}
