package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/applications/internal/store"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`,
		"user-"+t.Name()).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func insertJob(t *testing.T, pool *pgxpool.Pool, title string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, location, url, company_slug, source, updated_at)
		 VALUES ($1, 'Remote', $2, 'acme', 'greenhouse', NOW()) RETURNING id`,
		title, "https://example.com/"+t.Name()+"/"+title).Scan(&id)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func TestCreateApplication(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Engineer")

	app, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{
		JobID:     jobID,
		Notes:     "referred by a friend",
		AppliedAt: fp.Some("2026-01-02"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.UserID != userID || app.JobID != jobID || app.Notes != "referred by a friend" {
		t.Fatalf("app = %+v", app)
	}

	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); !errors.Is(err, store.ErrApplicationExists) {
		t.Fatalf("duplicate create: err = %v, want ErrApplicationExists", err)
	}
}

func TestListApplicationsByUserJoinsJobDetails(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Staff Engineer")

	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatal(err)
	}

	got, err := st.ListApplicationsByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].JobTitle != "Staff Engineer" || got[0].JobID != jobID {
		t.Fatalf("got = %+v", got)
	}
}

func TestListApplicationsByUserAndStatus(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	matching := insertJob(t, pool, "Match")
	other := insertJob(t, pool, "Other")

	status, err := st.CreateApplicationStatus(context.Background(), userID, "Applied", "#6366f1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: matching, StatusID: status.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: other}); err != nil {
		t.Fatal(err)
	}

	got, err := st.ListApplicationsByUserAndStatus(context.Background(), userID, status.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].JobID != matching {
		t.Fatalf("got = %+v", got)
	}
}

func TestUpdateApplication(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Engineer")
	created, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := st.UpdateApplication(context.Background(), userID, created.ID, dto.UpdateApplicationInput{Notes: "followed up"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Notes != "followed up" {
		t.Fatalf("notes = %q, want %q", updated.Notes, "followed up")
	}

	if _, err := st.UpdateApplication(context.Background(), userID, "00000000-0000-0000-0000-000000000000", dto.UpdateApplicationInput{}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing update: err = %v, want ErrNotFound", err)
	}
}

func TestDeleteApplication(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Engineer")
	created, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID})
	if err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteApplication(context.Background(), userID, created.ID); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListApplicationsByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty after delete", got)
	}
}

func TestGetApplicationsForJobs(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Engineer")
	otherJobID := insertJob(t, pool, "Other")
	created, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID})
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.GetApplicationsForJobs(context.Background(), userID, []string{jobID, otherJobID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[jobID].ApplicationID != created.ID {
		t.Fatalf("got = %+v", got)
	}
}

func TestSeedDefaultStatuses(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)

	if err := st.SeedDefaultStatuses(context.Background(), userID); err != nil {
		t.Fatal(err)
	}

	got, err := st.ListApplicationStatusesByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d default statuses, want 5", len(got))
	}
}

func TestApplicationStatusCRUD(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)

	created, err := st.CreateApplicationStatus(context.Background(), userID, "Offer", "#22c55e")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := st.UpdateApplicationStatus(context.Background(), created.ID, userID, "Offer!", "#22c55e")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Offer!" {
		t.Fatalf("name = %q, want Offer!", updated.Name)
	}

	if err := st.DeleteApplicationStatus(context.Background(), created.ID, userID); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListApplicationStatusesByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty after delete", got)
	}
}

func TestCountApplicationsUsingStatus(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "Engineer")
	status, err := st.CreateApplicationStatus(context.Background(), userID, "Applied", "#6366f1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateApplication(context.Background(), userID, dto.CreateApplicationInput{JobID: jobID, StatusID: status.ID}); err != nil {
		t.Fatal(err)
	}

	count, err := st.CountApplicationsUsingStatus(context.Background(), status.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
