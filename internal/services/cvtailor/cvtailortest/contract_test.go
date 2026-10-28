package cvtailortest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	cvtailortest.RunStoreContract(t, func(t *testing.T) cvtailortest.Fixture {
		t.Helper()
		store := cvtailortest.NewFakeStore()
		store.SetJob(dto.Job{ID: "job-1"})
		return cvtailortest.Fixture{
			Store: store, UserID: "user-1", Other: "user-2", JobID: "job-1",
			TrackLetterTab: func(_ *testing.T, userID, docID, tabID string) { store.TrackLetterTab(userID, docID, tabID) },
		}
	})
}
