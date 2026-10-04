package applicationstest_test

import (
	"fmt"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	applicationstest.RunStoreContract(t, func(t *testing.T) applicationstest.Fixture {
		t.Helper()
		var jobs int
		newJob := func() string {
			jobs++
			return fmt.Sprintf("job-extra-%d", jobs)
		}
		return applicationstest.Fixture{Store: applicationstest.NewFakeStore(), UserID: "user-1", JobID: "job-1", NewJob: newJob}
	})
}
