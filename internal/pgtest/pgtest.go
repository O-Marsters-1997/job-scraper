// Package pgtest provides a shared Postgres testcontainer for integration
// tests, one database per test (https://github.com/O-Marsters-1997/job-scraper/issues/932).
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	containerName  = "job-scraper-pgtest-v1"
	image          = "postgres:17-alpine"
	serverEnv      = "TEST_DATABASE_URL"
	maxConns       = 4
	idleSeconds    = 300
	startupTimeout = 2 * time.Minute
	dropTimeout    = 30 * time.Second
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

type server struct {
	admin    *pgxpool.Pool
	template string
}

var (
	once     sync.Once
	shared   *server
	startErr error
)

func New(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(tb.Context()), startupTimeout)
		defer cancel()
		shared, startErr = start(ctx)
	})
	if startErr != nil {
		tb.Fatalf("pgtest: %v", startErr)
	}

	pool, err := shared.create(tb)
	if err != nil {
		tb.Fatalf("pgtest: %v", err)
	}
	return pool
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

func start(ctx context.Context) (*server, error) {
	if err := chdirToRepoRoot(); err != nil {
		return nil, err
	}

	if raw := os.Getenv(serverEnv); raw != "" {
		return connect(ctx, raw)
	}

	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		return nil, fmt.Errorf("disable ryuk: %w", err)
	}
	var errs []error
	for range 2 {
		s, err := runAndConnect(ctx)
		if err == nil {
			return s, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("start postgres (needs Docker running, or %s set to a server URL): %w", serverEnv, errors.Join(errs...))
}

func runAndConnect(ctx context.Context) (*server, error) {
	connStr, err := runContainer(ctx)
	if err != nil {
		return nil, err
	}
	return connect(ctx, connStr)
}

func connect(ctx context.Context, connStr string) (*server, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}
	cfg.MaxConns = maxConns
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	s := &server{admin: admin}
	if err := s.prepare(ctx); err != nil {
		admin.Close()
		return nil, err
	}
	return s, nil
}

func (s *server) create(tb testing.TB) (*pgxpool.Pool, error) {
	tb.Helper()
	ctx := tb.Context()
	name := fmt.Sprintf("pgtest_%d_%08x", time.Now().Unix(), randomSuffix())
	if _, err := s.admin.Exec(ctx, "CREATE DATABASE "+ident(name)+" TEMPLATE "+ident(s.template)); err != nil {
		return nil, fmt.Errorf("create database %s: %w", name, err)
	}

	cfg := s.admin.Config()
	cfg.ConnConfig.Database = name
	cfg.MaxConns = maxConns
	cfg.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("postgres connect: %w", err), s.drop(ctx, name))
	}

	tb.Cleanup(func() {
		pool.Close()
		if err := s.drop(context.WithoutCancel(ctx), name); err != nil {
			tb.Errorf("pgtest: %v", err)
		}
	})
	return pool, nil
}

func randomSuffix() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint32(b[:])
}

func (s *server) drop(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, dropTimeout)
	defer cancel()
	if _, err := s.admin.Exec(ctx, "DROP DATABASE IF EXISTS "+ident(name)+" WITH (FORCE)"); err != nil {
		return fmt.Errorf("drop database %s: %w", name, err)
	}
	return nil
}

func runContainer(ctx context.Context) (string, error) {
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
