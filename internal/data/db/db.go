package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool connection pool.
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

	return &DB{pool: pool}, nil
}

// Close closes the connection pool.
func (db *DB) Close() {
	db.pool.Close()
}

// Pool returns the underlying connection pool for use by repository types.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}
