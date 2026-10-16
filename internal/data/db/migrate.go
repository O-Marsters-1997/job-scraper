package db

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"log/slog"
)

const migrationsDir = "scripts/migrations"

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, sqlDB, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	slog.InfoContext(ctx, "migrations applied")

	return nil
}

func CheckSchemaNotAhead(ctx context.Context, pool *pgxpool.Pool, bundled fs.FS) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	dbVersion, err := goose.GetDBVersionContext(ctx, sqlDB)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	buildVersion, err := newestBundled(bundled)
	if err != nil {
		return err
	}
	if dbVersion > buildVersion {
		return fmt.Errorf("schema version %d is ahead of this build's newest migration %d: deploy a newer build", dbVersion, buildVersion)
	}

	return nil
}

func newestBundled(bundled fs.FS) (int64, error) {
	names, err := fs.Glob(bundled, "*.sql")
	if err != nil {
		return 0, fmt.Errorf("list bundled migrations: %w", err)
	}
	var newest int64
	for _, name := range names {
		v, err := goose.NumericComponent(name)
		if err != nil {
			return 0, fmt.Errorf("parse migration %s: %w", name, err)
		}
		newest = max(newest, v)
	}

	return newest, nil
}
