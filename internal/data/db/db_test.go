package db_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

var testDB *db.DB

func TestMain(m *testing.M) {
	pool, err := pgtest.Pool()
	if err != nil {
		log.Fatalf("pgtest: %v", err)
	}
	testDB = db.NewFromPool(pool)

	os.Exit(m.Run())
}

func truncate(t testing.TB) {
	t.Helper()
	if _, err := testDB.Pool().Exec(context.Background(), "TRUNCATE jobs, scoring_options CASCADE"); err != nil {
		t.Fatalf("truncate jobs: %v", err)
	}
}
