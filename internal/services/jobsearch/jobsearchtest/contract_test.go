package jobsearchtest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	jobsearchtest.RunStoreContract(t, func(t *testing.T) (jobsearch.Store, string) {
		t.Helper()
		return jobsearchtest.NewFakeStore(), "contract-user"
	})
}
