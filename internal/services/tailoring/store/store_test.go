package store_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/tailoring/store"
	"github.com/ollymarsters/job-scraper/internal/services/tailoring/tailoringtest"
)

var seedCounter atomic.Int64

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	username := fmt.Sprintf("user-%s-%d", t.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`,
		username).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func TestStoreContract(t *testing.T) {
	tailoringtest.RunStoreContract(t, func(t *testing.T) tailoringtest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return tailoringtest.Fixture{Store: store.New(pool), UserID: insertUser(t, pool), Other: insertUser(t, pool)}
	})
}

func TestDeletePositionCascadesAchievementRows(t *testing.T) {
	pool := pgtest.New(t)
	st := store.New(pool)
	ctx := context.Background()
	uid := insertUser(t, pool)

	p, err := st.CreatePosition(ctx, uid, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateAchievement(ctx, uid, dto.AchievementInput{PositionID: p.ID, Text: "shipped"}); err != nil {
		t.Fatal(err)
	}
	if err := st.DeletePosition(ctx, uid, p.ID); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM achievements`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("achievements rows after position delete = %d, want 0", n)
	}
}
