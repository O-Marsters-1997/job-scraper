package pgtest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var seedCounter atomic.Int64

func InsertUser(tb testing.TB, pool *pgxpool.Pool) string {
	tb.Helper()
	var id string
	username := fmt.Sprintf("user-%s-%d", tb.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`, username).Scan(&id)
	if err != nil {
		tb.Fatalf("pgtest: insert user: %v", err)
	}
	return id
}

func InsertJob(tb testing.TB, pool *pgxpool.Pool, title, fingerprint string) string {
	tb.Helper()
	var id string
	url := fmt.Sprintf("https://example.com/%s/%d", tb.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, location, url, company_slug, source, updated_at, description, content_fingerprint)
		 VALUES ($1, 'Remote', $2, 'acme', 'greenhouse', NOW(), 'Build things in Go', $3) RETURNING id`,
		title, url, fingerprint).Scan(&id)
	if err != nil {
		tb.Fatalf("pgtest: insert job: %v", err)
	}
	return id
}
