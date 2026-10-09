package pgtest

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func adminConfig(t *testing.T) *pgxpool.Config {
	t.Helper()
	p, err := Pool()
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	cfg := p.Config()
	cfg.ConnConfig.Database = "postgres"
	return cfg
}

func connect(t *testing.T, cfg *pgx.ConnConfig, database string) *pgx.Conn {
	t.Helper()
	cfg = cfg.Copy()
	cfg.Database = database
	conn, err := pgx.ConnectConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("connect %s: %v", database, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func createScratch(t *testing.T, admin *pgx.Conn, created time.Time) string {
	t.Helper()
	name := fmt.Sprintf("pgtest_%d_%08x", created.Unix(), rand.Uint32())
	if _, err := admin.Exec(t.Context(), "CREATE DATABASE "+ident(name)); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+ident(name)+" WITH (FORCE)")
	})
	return name
}

func databaseExists(t *testing.T, admin *pgx.Conn, name string) bool {
	t.Helper()
	var exists bool
	if err := admin.QueryRow(t.Context(), "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		t.Fatalf("find %s: %v", name, err)
	}
	return exists
}

func TestDropOrphans(t *testing.T) {
	cfg := adminConfig(t)
	admin := connect(t, cfg.ConnConfig, "postgres")
	now := time.Now()

	idleOld := createScratch(t, admin, now.Add(-2*orphanAge))
	liveOld := createScratch(t, admin, now.Add(-2*orphanAge))
	idleNew := createScratch(t, admin, now)
	connect(t, cfg.ConnConfig, liveOld)

	if err := dropOrphans(t.Context(), admin, now); err != nil {
		t.Fatalf("dropOrphans: %v", err)
	}

	tests := []struct {
		name     string
		database string
		want     bool
	}{
		{name: "drops an idle database older than a minute", database: idleOld, want: false},
		{name: "spares an old database with a connection", database: liveOld, want: true},
		{name: "spares an idle database younger than a minute", database: idleNew, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := databaseExists(t, admin, tt.database); got != tt.want {
				t.Errorf("exists(%s) after dropOrphans = %v, want %v", tt.database, got, tt.want)
			}
		})
	}
}

func TestCreateDatabase(t *testing.T) {
	cfg := adminConfig(t)
	admin := connect(t, cfg.ConnConfig, "postgres")

	first, err := createDatabase(t.Context(), cfg)
	if err != nil {
		t.Fatalf("createDatabase: %v", err)
	}
	second, err := createDatabase(t.Context(), cfg)
	if err != nil {
		t.Fatalf("createDatabase: %v", err)
	}
	t.Cleanup(func() {
		for _, name := range []string{first, second} {
			_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+ident(name)+" WITH (FORCE)")
		}
	})

	t.Run("names each database uniquely", func(t *testing.T) {
		if first == second {
			t.Errorf("createDatabase returned %s twice, want distinct names", first)
		}
		for _, name := range []string{first, second} {
			if !binaryDatabase.MatchString(name) {
				t.Errorf("createDatabase = %s, want a name matching %s", name, binaryDatabase)
			}
		}
	})

	t.Run("clones a migrated database", func(t *testing.T) {
		var version int64
		if err := connect(t, cfg.ConnConfig, second).QueryRow(t.Context(), "SELECT max(version_id) FROM goose_db_version").Scan(&version); err != nil {
			t.Fatalf("query goose_db_version: %v", err)
		}
		if version == 0 {
			t.Errorf("version = 0, want migrations applied")
		}
	})

	t.Run("keeps one template per migrations hash", func(t *testing.T) {
		hash, err := migrationsHash()
		if err != nil {
			t.Fatalf("migrationsHash: %v", err)
		}
		var n int
		if err := admin.QueryRow(t.Context(), "SELECT count(*) FROM pg_database WHERE datname LIKE $1", "pgtest_tpl_"+hash+"%").Scan(&n); err != nil {
			t.Fatalf("count templates: %v", err)
		}
		if n != 1 {
			t.Errorf("templates for hash %s = %d, want 1", hash, n)
		}
	})
}
