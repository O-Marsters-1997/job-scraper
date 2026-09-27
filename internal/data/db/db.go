package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
)

// ScoringPort is scoring's tx-scoped facade, called from within this
// package's own transactions instead of writing scoring's tables directly
// (ADR 0011).
type ScoringPort interface {
	JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error
	JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error
	CompanyTracked(ctx context.Context, tx pgx.Tx, userID, companyID string) error
}

type DB struct {
	pool    *pgxpool.Pool
	queries *pgsqlc.Queries
	scoring ScoringPort
}

// WithScoring sets the scoring facade DB writes cross-context effects through.
func (db *DB) WithScoring(s ScoringPort) *DB {
	db.scoring = s
	return db
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

func NewFromPool(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool, queries: pgsqlc.New(pool)}
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return id.String()
}

// ConnString re-exports data.ConnString.
func ConnString() (string, error) {
	return data.ConnString()
}
