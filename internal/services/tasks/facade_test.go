package tasks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/taskstest"
)

type failingSeed struct {
	tasks.Store
	err error
}

func (f failingSeed) SeedDefaultTaskTypes(context.Context, pgx.Tx, string) error { return f.err }

func TestSeedDefaultsSeedsTheUsersTypes(t *testing.T) {
	t.Parallel()
	store := taskstest.NewFakeStore()
	m := tasks.Build(tasks.Deps{Store: store})

	if err := m.SeedDefaults(t.Context(), nil, userID); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListTaskTypes(t.Context(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("SeedDefaults seeded %d types, want 5", len(got))
	}
}

func TestSeedDefaultsPropagatesStoreError(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("boom")
	m := tasks.Build(tasks.Deps{Store: failingSeed{Store: taskstest.NewFakeStore(), err: wantErr}})

	if err := m.SeedDefaults(t.Context(), nil, userID); !errors.Is(err, wantErr) {
		t.Fatalf("SeedDefaults(...) err = %v, want %v", err, wantErr)
	}
}
