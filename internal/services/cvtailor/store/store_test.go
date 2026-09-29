package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
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

func insertJob(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, url, company_slug, source, updated_at, description, content_fingerprint)
		 VALUES ('Role', 'https://example.com/' || gen_random_uuid()::text, 'acme', 'test', NOW(), 'Build things in Go', 'fp-1')
		 RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func TestStoreDraftContract(t *testing.T) {
	cvtailortest.RunDraftContract(t, func(t *testing.T) cvtailortest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return cvtailortest.Fixture{
			Store: store.New(pool), UserID: insertUser(t, pool), Other: insertUser(t, pool), JobID: insertJob(t, pool),
		}
	})
}

func TestClaimDraftReclaimsAfterLeaseExpiry(t *testing.T) {
	pool := pgtest.New(t)
	st := store.New(pool)
	ctx := context.Background()
	uid, jobID := insertUser(t, pool), insertJob(t, pool)
	d, err := st.CreateDraft(ctx, uid, dto.DraftInput{JobID: jobID, DocID: "doc", TabID: "t.0", AchievementIDs: []string{"00000000-0000-0000-0000-00000000dead"}})
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.ClaimDraft(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.JobDescription != "Build things in Go" || first.JobFingerprint != "fp-1" {
		t.Errorf("ClaimDraft() job facts = %q, %q, want the Job's description and fingerprint", first.JobDescription, first.JobFingerprint)
	}
	if _, err := st.ClaimDraft(ctx); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("ClaimDraft() under a live lease err = %v, want ErrNotFound", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE tailored_cvs SET lease_until = NOW() - interval '1 second' WHERE id = $1`, d.ID); err != nil {
		t.Fatal(err)
	}
	second, err := st.ClaimDraft(ctx)
	if err != nil {
		t.Fatalf("ClaimDraft() after lease expiry err = %v, want the crashed Draft", err)
	}
	if second.ID != d.ID || second.Attempts != 2 {
		t.Errorf("ClaimDraft() = %+v, want Draft %s on attempt 2", second, d.ID)
	}
	if err := st.CompleteDraft(ctx, first, dto.DraftResult{EditSet: json.RawMessage(`{}`)}); err == nil {
		t.Error("CompleteDraft() by the crashed claim err = nil, want it rejected")
	}
}

func TestCreateDraftUnknownJobIsNotFound(t *testing.T) {
	pool := pgtest.New(t)
	_, err := store.New(pool).CreateDraft(context.Background(), insertUser(t, pool), dto.DraftInput{
		JobID: "00000000-0000-0000-0000-00000000dead", DocID: "doc", TabID: "t.0", AchievementIDs: []string{"00000000-0000-0000-0000-00000000dead"},
	})
	if !errors.Is(err, store.ErrJobNotFound) {
		t.Errorf("CreateDraft() err = %v, want ErrJobNotFound", err)
	}
}

func TestOnlyOneKeptTailoredCVPerJob(t *testing.T) {
	pool := pgtest.New(t)
	ctx := context.Background()
	uid, jobID := insertUser(t, pool), insertJob(t, pool)
	insert := `INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids, outcome)
		VALUES ($1, $2, 'doc', 't.0', '{}', $3)`
	if _, err := pool.Exec(ctx, insert, uid, jobID, "kept"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, insert, uid, jobID, "discarded"); err != nil {
		t.Errorf("second discarded row err = %v, want it allowed", err)
	}
	if _, err := pool.Exec(ctx, insert, uid, jobID, "kept"); err == nil {
		t.Error("second kept row err = nil, want the unique index to reject it")
	}
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
