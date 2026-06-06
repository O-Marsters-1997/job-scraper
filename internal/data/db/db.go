package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
)

type DB struct {
	pool    *pgxpool.Pool
	queries *pgsqlc.Queries
}

func New(ctx context.Context, connString string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	slog.Info("connected to postgres", slog.String("host", pool.Config().ConnConfig.Host))

	return &DB{pool: pool, queries: pgsqlc.New(pool)}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func ConnString() (string, error) {
	vars := map[string]string{
		"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
		"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
		"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
		"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
		"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
	}
	var missing []string
	for k, v := range vars {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		vars["POSTGRES_USER"], vars["POSTGRES_PASSWORD"],
		vars["POSTGRES_HOST"], vars["POSTGRES_PORT"], vars["POSTGRES_DB"], sslmode,
	), nil
}
