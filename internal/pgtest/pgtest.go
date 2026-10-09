// Package pgtest provides a shared Postgres testcontainer for integration
// tests, one database per test binary (https://github.com/O-Marsters-1997/job-scraper/issues/840).
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

	"github.com/docker/docker/api/types/container"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	containerName = "job-scraper-pgtest-v1"
	image         = "postgres:17-alpine"
	maxConns      = 4
	idleSeconds   = 300
)

// Ryuk reaps a reused container with its creator, so it stops itself when idle: https://golang.testcontainers.org/features/garbage_collector/
var entrypoint = fmt.Sprintf(`
(
	idle=0
	while sleep 10; do
		n=$(psql -U postgres -d postgres -tAc "SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend' AND pid <> pg_backend_pid()" 2>/dev/null) || { idle=0; continue; }
		if [ "$n" = 0 ]; then idle=$((idle + 10)); else idle=0; fi
		if [ "$idle" -ge %d ]; then kill -INT 1; fi
	done
) &
exec docker-entrypoint.sh "$@"
`, idleSeconds)

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

	adminConnStr, err := runContainer(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(adminConnStr)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	name, err := createDatabase(ctx, cfg.Copy())
	if err != nil {
		return nil, err
	}

	cfg.ConnConfig.Database = name
	cfg.MaxConns = maxConns
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	return pool, nil
}

func runContainer(ctx context.Context) (string, error) {
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		return "", fmt.Errorf("disable ryuk: %w", err)
	}

	c, err := postgres.Run(ctx, image,
		testcontainers.WithReuseByName(containerName),
		testcontainers.WithEntrypoint("sh", "-c", entrypoint, "pgtest"),
		testcontainers.WithCmd("postgres", "-c", "fsync=off", "-c", "max_connections=500"),
		testcontainers.WithTmpfs(map[string]string{"/var/lib/postgresql/data": "rw"}),
		testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) { hc.AutoRemove = true }),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return "", fmt.Errorf("start postgres container: %w", err)
	}

	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", fmt.Errorf("connection string: %w", err)
	}
	return connStr, nil
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
	ctx := tb.Context()

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
