package store_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
)

func newStore(t *testing.T) (st *store.Store, pool *pgxpool.Pool, userID string) {
	t.Helper()
	pool = pgtest.New(t)
	return store.New(pool), pool, pgtest.InsertUser(t, pool)
}

func TestStoreContract(t *testing.T) {
	applicationstest.RunStoreContract(t, func(t *testing.T) applicationstest.Fixture {
		t.Helper()
		st, pool, userID := newStore(t)
		return applicationstest.Fixture{
			Store:  st,
			UserID: userID,
			JobID:  pgtest.InsertJob(t, pool, "Contract Job", "Contract Job"),
		}
	})
}

func TestCreateApplicationRejectsDuplicate(t *testing.T) {
	st, pool, userID := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "Engineer")

	if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatalf("first CreateApplication err = %v", err)
	}
	_, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID})
	if !errors.Is(err, store.ErrApplicationExists) {
		t.Errorf("duplicate CreateApplication err = %v, want ErrApplicationExists", err)
	}
}

func TestListApplicationsJoinsJobDetails(t *testing.T) {
	st, pool, userID := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Staff Engineer", "Staff Engineer")
	if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatalf("CreateApplication err = %v", err)
	}

	got, err := st.ListApplications(t.Context(), userID, "")
	if err != nil {
		t.Fatalf("ListApplications err = %v", err)
	}
	if len(got) != 1 || got[0].JobTitle != "Staff Engineer" || got[0].JobID != jobID {
		t.Errorf("ListApplications = %+v, want one application for %q", got, "Staff Engineer")
	}
}

func TestListApplicationsFiltersByStatus(t *testing.T) {
	st, pool, userID := newStore(t)
	matching := pgtest.InsertJob(t, pool, "Match", "Match")
	other := pgtest.InsertJob(t, pool, "Other", "Other")
	status, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil)
	if err != nil {
		t.Fatalf("CreateApplicationStatus err = %v", err)
	}
	for _, in := range []dto.CreateApplicationInput{
		{JobID: matching, StatusID: status.ID},
		{JobID: other},
	} {
		if _, err := st.CreateApplication(t.Context(), userID, in); err != nil {
			t.Fatalf("CreateApplication(%+v) err = %v", in, err)
		}
	}

	got, err := st.ListApplications(t.Context(), userID, status.ID)
	if err != nil {
		t.Fatalf("ListApplications err = %v", err)
	}
	if len(got) != 1 || got[0].JobID != matching {
		t.Errorf("ListApplications(status) = %+v, want only the job %s", got, matching)
	}
}

func TestSeedDefaultStatuses(t *testing.T) {
	tests := []struct {
		name   string
		commit bool
		want   int
	}{
		{name: "commit keeps the defaults", commit: true, want: 5},
		{name: "rollback discards the defaults", commit: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, pool, userID := newStore(t)
			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				return st.SeedDefaultStatuses(t.Context(), tx, userID)
			})

			got, err := st.ListApplicationStatusesByUser(t.Context(), userID)
			if err != nil {
				t.Fatalf("ListApplicationStatusesByUser err = %v", err)
			}
			if len(got) != tt.want {
				t.Errorf("default statuses = %d, want %d", len(got), tt.want)
			}
		})
	}
}
