// Package pgtest provides a shared Postgres testcontainer for integration
// tests, one container per test binary.
package pgtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ollymarsters/job-scraper/internal/data/db"
)

var (
	once    sync.Once
	pool    *pgxpool.Pool
	initErr error
)

// Pool returns the pool shared by this test binary, starting the
// container and applying migrations on the first call.
func Pool() (*pgxpool.Pool, error) {
	once.Do(func() {
		pool, initErr = start(context.Background())
	})
	return pool, initErr
}

// New returns the pool shared by this test binary (see Pool), with
// every table truncated so the caller starts from an empty database.
func New(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	p, err := Pool()
	if err != nil {
		tb.Fatalf("pgtest: %v", err)
	}
	truncateAll(tb, p)

	return p
}

func InTx(tb testing.TB, pool *pgxpool.Pool, commit bool, fn func(pgx.Tx) error) {
	tb.Helper()
	ctx := tb.Context()

	tx, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("pgtest: begin: %v", err)
	}
	tb.Cleanup(func() { _ = tx.Rollback(context.WithoutCancel(ctx)) })

	if err := fn(tx); err != nil {
		tb.Fatalf("pgtest: tx body: %v", err)
	}
	if !commit {
		if err := tx.Rollback(ctx); err != nil {
			tb.Fatalf("pgtest: rollback: %v", err)
		}
		return
	}
	if err := tx.Commit(ctx); err != nil {
		tb.Fatalf("pgtest: commit: %v", err)
	}
}

func start(ctx context.Context) (*pgxpool.Pool, error) {
	if err := chdirToRepoRoot(); err != nil {
		return nil, err
	}

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	if err := db.RunMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return pool, nil
}

// chdirToRepoRoot resolves "scripts/migrations" against the repo root:
// goose reads it relative to the working directory, which go test sets
// to the package directory.
func chdirToRepoRoot() error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("pgtest: cannot determine caller")
	}
	return os.Chdir(filepath.Join(filepath.Dir(filename), "../.."))
}

func truncateAll(tb testing.TB, pool *pgxpool.Pool) {
	tb.Helper()
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name != 'goose_db_version'
	`)
	if err != nil {
		tb.Fatalf("pgtest: list tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			tb.Fatalf("pgtest: scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		tb.Fatalf("pgtest: list tables: %v", err)
	}
	if len(tables) == 0 {
		return
	}

	if _, err := pool.Exec(ctx, "TRUNCATE "+strings.Join(tables, ", ")+" CASCADE"); err != nil {
		tb.Fatalf("pgtest: truncate: %v", err)
	}
}
