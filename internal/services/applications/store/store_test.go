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
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses/applicationstatusestest"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`,
		"user-"+t.Name()).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func insertJob(t *testing.T, pool *pgxpool.Pool, title string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, location, url, company_slug, source, updated_at)
		 VALUES ($1, 'Remote', $2, 'acme', 'greenhouse', NOW()) RETURNING id`,
		title, "https://example.com/"+t.Name()+"/"+title).Scan(&id)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func TestApplicationsStoreContract(t *testing.T) {
	applicationstest.RunStoreContract(t, func(t *testing.T) applicationstest.Fixture {
		t.Helper()
		st, pool := newStore(t)
		return applicationstest.Fixture{
			Store:  st,
			UserID: insertUser(t, pool),
			JobID:  insertJob(t, pool, "Contract Job"),
		}
	})
}

func TestApplicationStatusesStoreContract(t *testing.T) {
	applicationstatusestest.RunStoreContract(t, func(t *testing.T) applicationstatusestest.Fixture {
		t.Helper()
		st, pool := newStore(t)
		return applicationstatusestest.Fixture{
			Store:  st,
			UserID: insertUser(t, pool),
		}
	})
}

func TestCreateApplicationRejectsDuplicate(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Engineer")

	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); !errors.Is(err, store.ErrApplicationExists) {
		t.Fatalf("duplicate create: err = %v, want ErrApplicationExists", err)
	}
}

func TestListApplicationsByUserJoinsJobDetails(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Staff Engineer")

	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatal(err)
	}

	got, err := st.ListApplicationsByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].JobTitle != "Staff Engineer" || got[0].JobID != jobID {
		t.Fatalf("got = %+v", got)
	}
}

func TestListApplicationsByUserAndStatus(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	matching := insertJob(t, pool, "Match")
	other := insertJob(t, pool, "Other")

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

	got, err := st.ListApplicationsByUserAndStatus(context.Background(), userID, status.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].JobID != matching {
		t.Fatalf("got = %+v", got)
	}
}

func TestSeedDefaultStatuses(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
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
	userID := insertUser(t, pool)
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
