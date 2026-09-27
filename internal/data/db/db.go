package db

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
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
		sort.Strings(missing)
		return "", fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	dbName := vars["POSTGRES_DB"]
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(vars["POSTGRES_USER"], vars["POSTGRES_PASSWORD"]),
		Host:     net.JoinHostPort(vars["POSTGRES_HOST"], vars["POSTGRES_PORT"]),
		Path:     "/" + dbName,
		RawPath:  "/" + url.PathEscape(dbName),
		RawQuery: url.Values{"sslmode": {sslmode}}.Encode(),
	}).String(), nil
}
