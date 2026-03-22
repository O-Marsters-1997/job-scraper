package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

// New opens a connection pool to Postgres and verifies connectivity with Ping.
// connString format: "postgres://user:pass@host:port/dbname"
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

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

// Pool returns the underlying connection pool for use by repository types.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// ConnString builds a Postgres DSN from individual env vars:
// POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB.
func ConnString() string {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbname := os.Getenv("POSTGRES_DB")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
}
