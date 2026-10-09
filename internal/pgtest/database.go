package pgtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
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
	templateLockKey = 840
	orphanAge       = time.Minute
)

var binaryDatabase = regexp.MustCompile(`^pgtest_([0-9]+)_[0-9a-f]+$`)

func createDatabase(ctx context.Context, admin *pgxpool.Config) (string, error) {
	conn, err := pgx.ConnectConfig(ctx, admin.ConnConfig)
	if err != nil {
		return "", fmt.Errorf("postgres connect: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	now := time.Now()
	if err := dropOrphans(ctx, conn, now); err != nil {
		return "", err
	}

	hash, err := migrationsHash()
	if err != nil {
		return "", err
	}
	template := "pgtest_tpl_" + hash

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", templateLockKey); err != nil {
		return "", fmt.Errorf("lock template: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", templateLockKey) }()

	if err := ensureTemplate(ctx, conn, admin, template); err != nil {
		return "", err
	}

	name := fmt.Sprintf("pgtest_%d_%08x", now.Unix(), rand.Uint32())
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+ident(name)+" TEMPLATE "+ident(template)); err != nil {
		return "", fmt.Errorf("create database %s: %w", name, err)
	}
	return name, nil
}

func ensureTemplate(ctx context.Context, conn *pgx.Conn, admin *pgxpool.Config, template string) error {
	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", template).Scan(&exists); err != nil {
		return fmt.Errorf("find template: %w", err)
	}
	if exists {
		return nil
	}

	build := template + "_build"
	if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+ident(build)+" WITH (FORCE)"); err != nil {
		return fmt.Errorf("drop stale template build: %w", err)
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+ident(build)); err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	if err := migrate(ctx, admin, build); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, "ALTER DATABASE "+ident(build)+" RENAME TO "+ident(template)); err != nil {
		return fmt.Errorf("publish template: %w", err)
	}
	return nil
}

func migrate(ctx context.Context, admin *pgxpool.Config, database string) error {
	cfg := admin.Copy()
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

func dropOrphans(ctx context.Context, conn *pgx.Conn, now time.Time) error {
	rows, err := conn.Query(ctx, `
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
		m := binaryDatabase.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		created, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || now.Sub(time.Unix(created, 0)) < orphanAge {
			continue
		}
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+ident(name)+" WITH (FORCE)"); err != nil {
			return fmt.Errorf("drop orphan %s: %w", name, err)
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
		fmt.Fprintf(h, "%s\x00%d\x00", e.Name(), len(body))
		h.Write(body)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func ident(name string) string {
	return pgx.Identifier{name}.Sanitize()
}
