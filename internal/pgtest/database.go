package pgtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data/db"
)

const (
	migrationsDir   = "scripts/migrations"
	templateLockKey = 932
	staleAfter      = 5 * time.Minute
)

var testDatabase = regexp.MustCompile(`^pgtest_([0-9]+)_[0-9a-f]{8}$`)

func (s *server) prepare(ctx context.Context) error {
	if err := s.ping(ctx); err != nil {
		return err
	}
	if err := s.sweep(ctx, time.Now()); err != nil {
		return err
	}
	template, err := s.ensureTemplate(ctx)
	if err != nil {
		return err
	}
	s.template = template
	return nil
}

// A reused container can pass the log wait before accepting connections: https://github.com/testcontainers/testcontainers-go/issues/3671
func (s *server) ping(ctx context.Context) error {
	for {
		_, err := s.admin.Exec(ctx, "SELECT 1")
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping server: %w", err)
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (s *server) ensureTemplate(ctx context.Context) (template string, err error) {
	hash, err := migrationsHash()
	if err != nil {
		return "", err
	}
	template = "pgtest_tpl_" + hash

	conn, err := s.admin.Acquire(ctx)
	if err != nil {
		return "", fmt.Errorf("acquire admin connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", templateLockKey); err != nil {
		return "", fmt.Errorf("lock template: %w", err)
	}
	defer func() {
		if _, uerr := conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", templateLockKey); uerr != nil {
			err = errors.Join(err, fmt.Errorf("unlock template: %w", uerr))
		}
	}()

	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", template).Scan(&exists); err != nil {
		return "", fmt.Errorf("find template: %w", err)
	}
	if exists {
		return template, nil
	}

	build := template + "_build"
	if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+ident(build)+" WITH (FORCE)"); err != nil {
		return "", fmt.Errorf("drop stale template build: %w", err)
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+ident(build)); err != nil {
		return "", fmt.Errorf("create template: %w", err)
	}
	if err := s.migrate(ctx, build); err != nil {
		return "", err
	}
	if _, err := conn.Exec(ctx, "ALTER DATABASE "+ident(build)+" RENAME TO "+ident(template)); err != nil {
		return "", fmt.Errorf("publish template: %w", err)
	}
	return template, nil
}

func (s *server) migrate(ctx context.Context, database string) error {
	cfg := s.admin.Config()
	cfg.ConnConfig.Database = database
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (s *server) sweep(ctx context.Context, now time.Time) error {
	rows, err := s.admin.Query(ctx, `
		SELECT datname FROM pg_database d
		WHERE NOT EXISTS (SELECT 1 FROM pg_stat_activity a WHERE a.datname = d.datname)
	`)
	if err != nil {
		return fmt.Errorf("list databases: %w", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("list databases: %w", err)
	}

	for _, name := range names {
		m := testDatabase.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		created, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || now.Sub(time.Unix(created, 0)) < staleAfter {
			continue
		}
		if err := s.drop(name); err != nil {
			return fmt.Errorf("sweep: %w", err)
		}
	}
	return nil
}

func migrationsHash() (string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return "", fmt.Errorf("hash migrations: %w", err)
	}
	h := sha256.New()
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			return "", fmt.Errorf("hash migrations: %w", err)
		}
		h.Write(fmt.Appendf(nil, "%s\x00%d\x00", e.Name(), len(body)))
		h.Write(body)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func ident(name string) string {
	return pgx.Identifier{name}.Sanitize()
}
