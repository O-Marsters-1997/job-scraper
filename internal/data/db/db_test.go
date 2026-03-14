package db_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/db"
)

var testDB *db.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	// RunMigrations resolves "scripts/migrations" relative to CWD.
	// go test sets CWD to the package directory; navigate up to the module root.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot determine test file path")
	}
	moduleRoot := filepath.Join(filepath.Dir(filename), "../../..")
	if err := os.Chdir(moduleRoot); err != nil {
		log.Fatalf("chdir to module root: %v", err)
	}

	pgCont, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgCont.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	testDB, err = db.New(ctx, connStr)
	if err != nil {
		log.Fatalf("db.New: %v", err)
	}

	if err := data.RunMigrations(ctx, testDB.Pool()); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	code := m.Run()

	testDB.Close()
	if err := pgCont.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}

	os.Exit(code)
}

// truncate clears the jobs table so each test starts from a clean state.
func truncate(t *testing.T) {
	t.Helper()
	if _, err := testDB.Pool().Exec(context.Background(), "TRUNCATE jobs"); err != nil {
		t.Fatalf("truncate jobs: %v", err)
	}
}
