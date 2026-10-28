package taskstest_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/taskstest"
)

func TestFakeStoreSatisfiesContract(t *testing.T) {
	t.Parallel()
	taskstest.RunStoreContract(t, func(t *testing.T) taskstest.Fixture {
		t.Helper()
		f := taskstest.NewFakeStore()
		return taskstest.Fixture{
			Store:        f,
			UserID:       "user-1",
			OtherUserID:  "user-2",
			TypeID:       f.AddType("user-1", "Apply").ID,
			OtherTypeID:  f.AddType("user-2", "Apply").ID,
			SystemTypeID: f.AddSystemType("user-1", "Submit", dto.TaskTypeApply).ID,
			JobID:        "job-1",
		}
	})
}
