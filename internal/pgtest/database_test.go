package pgtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func adminServer(t *testing.T) *server {
	t.Helper()
	New(t)
	return shared
}

func createScratch(t *testing.T, s *server, created time.Time) string {
	t.Helper()
	name := fmt.Sprintf("pgtest_%d_%08x", created.Unix(), randomSuffix())
	if _, err := s.admin.Exec(t.Context(), "CREATE DATABASE "+ident(name)); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	t.Cleanup(func() { _ = s.drop(context.WithoutCancel(t.Context()), name) })
	return name
}

func databaseExists(t *testing.T, s *server, name string) bool {
	t.Helper()
	var exists bool
	if err := s.admin.QueryRow(t.Context(), "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		t.Fatalf("find %s: %v", name, err)
	}
	return exists
}

func TestSweep(t *testing.T) {
	t.Parallel()
	s := adminServer(t)
	now := time.Now()

	idleOld := createScratch(t, s, now.Add(-2*staleAfter))
	liveOld := createScratch(t, s, now.Add(-2*staleAfter))
	idleNew := createScratch(t, s, now)

	cfg := s.admin.Config().ConnConfig.Copy()
	cfg.Database = liveOld
	conn, err := pgx.ConnectConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("connect %s: %v", liveOld, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(t.Context())) })

	if err := s.sweep(t.Context(), now); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	tests := []struct {
		name     string
		database string
		want     bool
	}{
		{name: "drops an idle database older than the stale age", database: idleOld, want: false},
		{name: "spares an old database with a connection", database: liveOld, want: true},
		{name: "spares an idle database younger than the stale age", database: idleNew, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := databaseExists(t, s, tt.database); got != tt.want {
				t.Errorf("exists(%s) after sweep = %v, want %v", tt.database, got, tt.want)
			}
		})
	}
}

func TestEnsureTemplate(t *testing.T) {
	t.Parallel()
	s := adminServer(t)

	got, err := s.ensureTemplate(t.Context())
	if err != nil {
		t.Fatalf("ensureTemplate: %v", err)
	}
	if got != s.template {
		t.Errorf("ensureTemplate = %s, want the existing template %s", got, s.template)
	}

	hash, err := migrationsHash()
	if err != nil {
		t.Fatalf("migrationsHash: %v", err)
	}
	var n int
	if err := s.admin.QueryRow(t.Context(), "SELECT count(*) FROM pg_database WHERE datname LIKE 'pgtest\\_tpl\\_' || $1 || '%'", hash).Scan(&n); err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if n != 1 {
		t.Errorf("templates for hash %s = %d, want 1", hash, n)
	}
}
