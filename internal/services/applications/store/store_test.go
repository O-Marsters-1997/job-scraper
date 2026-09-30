package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

func TestStoreContract(t *testing.T) {
	applicationstest.RunStoreContract(t, func(t *testing.T) applicationstest.Fixture {
		t.Helper()
		st, pool := newStore(t)
		return applicationstest.Fixture{
			Store:  st,
			UserID: pgtest.InsertUser(t, pool),
			JobID:  pgtest.InsertJob(t, pool, "Contract Job", "Contract Job"),
		}
	})
}

func TestCreateApplicationRejectsDuplicate(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "Engineer")

	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); !errors.Is(err, store.ErrApplicationExists) {
		t.Fatalf("duplicate create: err = %v, want ErrApplicationExists", err)
	}
}

func TestListApplicationsJoinsJobDetails(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Staff Engineer", "Staff Engineer")

	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatal(err)
	}

	got, err := st.ListApplications(context.Background(), userID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].JobTitle != "Staff Engineer" || got[0].JobID != jobID {
		t.Fatalf("got = %+v", got)
	}
}

func TestListApplicationsFiltersByStatus(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	matching := pgtest.InsertJob(t, pool, "Match", "Match")
	other := pgtest.InsertJob(t, pool, "Other", "Other")

	status, err := st.CreateApplicationStatus(context.Background(), userID, "Applied", "#6366f1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: matching, StatusID: status.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: other}); err != nil {
		t.Fatal(err)
	}

	got, err := st.ListApplications(context.Background(), userID, status.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].JobID != matching {
		t.Fatalf("got = %+v", got)
	}
}

func TestSeedDefaultStatuses(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := st.SeedDefaultStatuses(ctx, tx, userID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got, err := st.ListApplicationStatusesByUser(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d default statuses, want 5", len(got))
	}
}

func TestSeedDefaultStatusesRollsBackWithItsTx(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := st.SeedDefaultStatuses(ctx, tx, userID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	got, err := st.ListApplicationStatusesByUser(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d default statuses after rollback, want 0", len(got))
	}
}
