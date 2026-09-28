package applications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses"
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses/applicationstatusestest"
)

type failingSeed struct {
	applicationstatuses.Store
	err error
}

func (f failingSeed) SeedDefaultStatuses(context.Context, pgx.Tx, string) error { return f.err }

func TestSeedDefaultsSeedsTheUsersStatuses(t *testing.T) {
	statuses := applicationstatusestest.NewFakeStore()
	m := applications.Build(applications.Deps{
		Applications: applicationstest.NewFakeStore(),
		Statuses:     statuses,
	})

	if err := m.SeedDefaults(context.Background(), nil, "user-1"); err != nil {
		t.Fatal(err)
	}

	got, err := statuses.ListApplicationStatusesByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("SeedDefaults seeded %d statuses, want 5", len(got))
	}
}

func TestSeedDefaultsPropagatesStoreError(t *testing.T) {
	wantErr := errors.New("boom")
	m := applications.Build(applications.Deps{
		Applications: applicationstest.NewFakeStore(),
		Statuses:     failingSeed{Store: applicationstatusestest.NewFakeStore(), err: wantErr},
	})

	if err := m.SeedDefaults(context.Background(), nil, "user-1"); !errors.Is(err, wantErr) {
		t.Fatalf("SeedDefaults(...) err = %v, want %v", err, wantErr)
	}
}
