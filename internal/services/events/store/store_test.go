package store_test

import (
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/events/eventstest"
	"github.com/ollymarsters/job-scraper/internal/services/events/store"
)

func TestStoreContract(t *testing.T) {
	eventstest.RunStoreContract(t, func(t *testing.T) eventstest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return eventstest.Fixture{
			Store:       store.New(pool),
			UserID:      pgtest.InsertUser(t, pool),
			OtherUserID: pgtest.InsertUser(t, pool),
			SubjectID:   pgtest.InsertJob(t, pool, "Contract Job", "Contract Job"),
		}
	})
}

func TestInsertEventRollsBackWithItsTransaction(t *testing.T) {
	pool := pgtest.New(t)
	st := store.New(pool)
	userID := pgtest.InsertUser(t, pool)

	pgtest.InTx(t, pool, false, func(tx pgx.Tx) error {
		return st.InsertEvent(t.Context(), tx, userID, events.JobOpened, "", []byte(`{}`))
	})

	got, err := st.ListEvents(t.Context(), userID, "")
	if err != nil || len(got) != 0 {
		t.Errorf("ListEvents() after rollback = %+v, %v, want none", got, err)
	}
}
