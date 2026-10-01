package pgtest_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

type fatalTB struct {
	testing.TB
	cleanups []func()
}

func (f *fatalTB) Helper()                  {}
func (f *fatalTB) Cleanup(fn func())        { f.cleanups = append(f.cleanups, fn) }
func (f *fatalTB) Fatalf(string, ...any)    { runtime.Goexit() }
func (f *fatalTB) Context() context.Context { return context.Background() }

func openTxCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), "SELECT count(*) FROM pg_stat_activity WHERE state = 'idle in transaction'").Scan(&n)
	if err != nil {
		t.Fatalf("count open transactions: %v", err)
	}
	return n
}

func TestInTx(t *testing.T) {
	const insert = "INSERT INTO companies (slug, name) VALUES ('intx-check', 'Intx Check')"

	tests := []struct {
		name   string
		commit bool
		want   int
	}{
		{name: "commit keeps the rows", commit: true, want: 1},
		{name: "rollback discards the rows", commit: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := pgtest.New(t)
			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				_, err := tx.Exec(t.Context(), insert)
				return err
			})

			var got int
			if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM companies WHERE slug = 'intx-check'").Scan(&got); err != nil {
				t.Fatalf("count: %v", err)
			}
			if got != tt.want {
				t.Errorf("rows after InTx(commit=%v) = %d, want %d", tt.commit, got, tt.want)
			}
		})
	}

	t.Run("fatal inside fn leaves no open transaction", func(t *testing.T) {
		pool := pgtest.New(t)
		ftb := &fatalTB{TB: t}

		done := make(chan struct{})
		go func() {
			defer close(done)
			pgtest.InTx(ftb, pool, true, func(tx pgx.Tx) error {
				if _, err := tx.Exec(t.Context(), insert); err != nil {
					return err
				}
				ftb.Fatalf("boom")
				return nil
			})
		}()
		<-done

		if openTxCount(t, pool) == 0 {
			t.Fatalf("no open transaction before cleanup, test does not exercise the leak")
		}
		for _, fn := range ftb.cleanups {
			fn()
		}
		if got := openTxCount(t, pool); got != 0 {
			t.Errorf("open transactions after cleanup = %d, want 0", got)
		}
		pgtest.New(t)
	})
}

func TestNew(t *testing.T) {
	t.Run("applies migrations", func(t *testing.T) {
		pool := pgtest.New(t)

		var version int64
		if err := pool.QueryRow(t.Context(), "SELECT max(version_id) FROM goose_db_version").Scan(&version); err != nil {
			t.Fatalf("query goose_db_version: %v", err)
		}
		if version == 0 {
			t.Errorf("version = 0, want migrations applied")
		}
	})

	t.Run("returns the same pool across calls", func(t *testing.T) {
		first := pgtest.New(t)
		second := pgtest.New(t)
		if first != second {
			t.Errorf("New returned different pools across calls, want the same shared pool")
		}
	})

	t.Run("truncates tables between calls", func(t *testing.T) {
		ctx := t.Context()
		pool := pgtest.New(t)

		if _, err := pool.Exec(ctx, "INSERT INTO companies (slug, name) VALUES ('pgtest-check', 'Pgtest Check')"); err != nil {
			t.Fatalf("insert: %v", err)
		}

		pool = pgtest.New(t)

		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM companies WHERE slug = 'pgtest-check'").Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0 after truncation", count)
		}
	})
}
