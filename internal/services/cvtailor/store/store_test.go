package store_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
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
	cvtailortest.RunStoreContract(t, func(t *testing.T) cvtailortest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return cvtailortest.Fixture{Store: store.New(pool), UserID: insertUser(t, pool), Other: insertUser(t, pool)}
	})
}

func TestStoreHeadingMappingContract(t *testing.T) {
	cvtailortest.RunHeadingMappingContract(t, func(t *testing.T) cvtailortest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return cvtailortest.Fixture{Store: store.New(pool), UserID: insertUser(t, pool), Other: insertUser(t, pool)}
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

func TestImportPositionsIsAllOrNothing(t *testing.T) {
	pool := pgtest.New(t)
	st := store.New(pool)
	ctx := context.Background()
	uid := insertUser(t, pool)

	bad := "not-a-date"
	_, err := st.ImportPositions(ctx, uid, []dto.ImportPosition{
		{Employer: "Broken", Title: "Engineer", StartDate: &bad},
		{Employer: "Acme", Title: "Engineer", Achievements: []string{"shipped"}},
	})
	if err == nil {
		t.Fatal("ImportPositions() error = nil, want an error for the bad date")
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM positions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("positions rows after failed import = %d, want 0", n)
	}
}
