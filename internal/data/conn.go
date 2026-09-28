package data

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect builds a pgxpool from the POSTGRES_* env vars and pings it,
// shared by every binary that connects to the database directly.
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	connStr, err := ConnString()
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	slog.Info("connected to postgres", slog.String("host", pool.Config().ConnConfig.Host))
	return pool, nil
}

// ConnString builds the Postgres DSN from POSTGRES_* env vars, shared by
// every binary that connects to the database directly.
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
