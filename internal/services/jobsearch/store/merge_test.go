package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

const (
	libertyLinkedIn  = "https://www.linkedin.com/jobs/view/4472707630"
	libertyWTTJ      = "https://app.welcometothejungle.com/jobs/4LsTTZd3"
	libertyOtherRole = "https://app.welcometothejungle.com/jobs/4EP20yW7"
	heidiLinkedIn    = "https://www.linkedin.com/jobs/view/4474766548"
	heidiWIS         = "https://workinstartups.com/details/5909695355"
)

func loadListing(t *testing.T, url string) dto.Job {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "testdata", "cross-source-dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Source          string  `json:"source"`
		Title           string  `json:"title"`
		CompanySlug     string  `json:"company_slug"`
		Location        string  `json:"location"`
		URL             string  `json:"url"`
		PostingID       *string `json:"provider_posting_id"`
		SalaryRaw       string  `json:"salary_raw"`
		WorkArrangement string  `json:"work_arrangement"`
		Description     string  `json:"description"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.URL != url {
			continue
		}
		job := dto.Job{
			Title: r.Title, Location: r.Location, URL: r.URL, CompanySlug: r.CompanySlug,
			Source: r.Source, UpdatedAt: time.Now(), Description: r.Description,
			SalaryRaw: r.SalaryRaw, WorkArrangement: r.WorkArrangement,
		}
		if r.PostingID != nil {
			job.ProviderPostingID = *r.PostingID
		}
		return job
	}
	t.Fatalf("fixture has no listing %s", url)
	return dto.Job{}
}

func listingURLs(t *testing.T, pool *pgxpool.Pool, jobID string) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(), "SELECT normalized_url FROM job_urls WHERE job_id = $1 ORDER BY normalized_url", jobID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			t.Fatal(err)
		}
		urls = append(urls, u)
	}
	return urls
}

func jobCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM jobs").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSaveCanonicalMergesAggregatorListings(t *testing.T) {
	t.Run("a second aggregator listing joins the first Job", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		trackCompany(t, st, userID, upsertCompany(t, st, dto.CompanyUpsert{Slug: "heidi", Name: "Heidi"}).ID, 360)
		first, status := saveJob(t, st, loadListing(t, heidiLinkedIn))
		if status != "new" {
			t.Fatalf("first status = %q, want new", status)
		}
		effects := effectCountForJob(t, pool, first.ID)

		second, status := saveJob(t, st, loadListing(t, heidiWIS))
		if status != "merged" || second.ID != first.ID {
			t.Fatalf("second: status = %q, id = %q, want merged with id %q", status, second.ID, first.ID)
		}
		if got := effectCountForJob(t, pool, first.ID); got != effects {
			t.Errorf("effects after merge = %d, want %d", got, effects)
		}
		if diff := cmp.Diff([]string{heidiWIS, heidiLinkedIn}, listingURLs(t, pool, first.ID)); diff != "" {
			t.Errorf("job_urls (-want +got):\n%s", diff)
		}
		if n := jobCount(t, pool); n != 1 {
			t.Fatalf("jobs = %d, want 1", n)
		}
		want := loadListing(t, heidiLinkedIn)
		var title, description, url string
		err := pool.QueryRow(t.Context(), "SELECT title, description, url FROM jobs WHERE id = $1", first.ID).Scan(&title, &description, &url)
		if err != nil {
			t.Fatal(err)
		}
		if title != want.Title || description != want.Description || url != heidiLinkedIn {
			t.Errorf("job content changed by merge: title = %q, url = %q", title, url)
		}
	})

	t.Run("re-ingesting a secondary listing with new content changes nothing", func(t *testing.T) {
		st, pool, _ := newUserStore(t)
		first, _ := saveJob(t, st, loadListing(t, heidiLinkedIn))
		wis := loadListing(t, heidiWIS)
		saveJob(t, st, wis)

		wis.Description = "rewritten by the aggregator"
		got, status := saveJob(t, st, wis)
		if status != "unchanged" || got.ID != first.ID {
			t.Fatalf("status = %q, id = %q, want unchanged with id %q", status, got.ID, first.ID)
		}
		var description string
		if err := pool.QueryRow(t.Context(), "SELECT description FROM jobs WHERE id = $1", first.ID).Scan(&description); err != nil {
			t.Fatal(err)
		}
		if description != loadListing(t, heidiLinkedIn).Description {
			t.Errorf("job description = %q, want the first listing's", description)
		}
	})

	t.Run("a listing matching two open Jobs inserts a new Job", func(t *testing.T) {
		st, pool, _ := newUserStore(t)
		for url, location := range map[string]string{"https://a.example.com/1": "Paris", "https://b.example.com/1": "London"} {
			job := loadListing(t, heidiLinkedIn)
			job.URL, job.Location = url, location
			saveJob(t, st, job)
		}
		if n := jobCount(t, pool); n != 2 {
			t.Fatalf("setup: jobs = %d, want 2", n)
		}

		anywhere := loadListing(t, heidiWIS)
		anywhere.Location = ""
		if _, status := saveJob(t, st, anywhere); status != "new" {
			t.Errorf("status = %q, want new", status)
		}
		if n := jobCount(t, pool); n != 3 {
			t.Errorf("jobs = %d, want 3", n)
		}
	})

	for _, tt := range []struct{ name, set string }{
		{"closed", "closed_at = NOW()"},
		{"outside the 90-day window", "updated_at = NOW() - INTERVAL '91 days'"},
	} {
		t.Run("a "+tt.name+" candidate is not merged into", func(t *testing.T) {
			st, pool, _ := newUserStore(t)
			first, _ := saveJob(t, st, loadListing(t, heidiLinkedIn))
			if _, err := pool.Exec(t.Context(), "UPDATE jobs SET "+tt.set+" WHERE id = $1", first.ID); err != nil {
				t.Fatal(err)
			}
			second, status := saveJob(t, st, loadListing(t, heidiWIS))
			if status != "new" || second.ID == first.ID {
				t.Errorf("status = %q, id = %q, want new with an id other than %q", status, second.ID, first.ID)
			}
		})
	}
}

func libertyBoard(t *testing.T, st *store.Store) (boardID, companyID string) {
	t.Helper()
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "liberty-global", Name: "Liberty Global"})
	board, err := st.UpsertCandidateBoard(t.Context(), company.ID, "wttj", "liberty-global")
	if err != nil {
		t.Fatalf("UpsertCandidateBoard() err = %v", err)
	}
	return board.ID, company.ID
}

func atsListing(t *testing.T, url, boardID, companyID string) dto.Job {
	t.Helper()
	job := loadListing(t, url)
	job.BoardID, job.CompanyID = boardID, companyID
	return job
}

func TestSaveCanonicalUpgradesAggregatorJob(t *testing.T) {
	t.Run("the ATS posting upgrades the aggregator Job in place", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		boardID, companyID := libertyBoard(t, st)
		trackCompany(t, st, userID, companyID, 60)
		first, _ := saveJob(t, st, loadListing(t, libertyLinkedIn))
		wttj := atsListing(t, libertyWTTJ, boardID, companyID)

		got, status := saveJob(t, st, wttj)
		if status != "upgraded" || got.ID != first.ID {
			t.Fatalf("status = %q, id = %q, want upgraded with id %q", status, got.ID, first.ID)
		}
		if n := jobCount(t, pool); n != 1 {
			t.Fatalf("jobs = %d, want 1", n)
		}
		var title, description, url, source, posting string
		err := pool.QueryRow(t.Context(), "SELECT title, description, url, source, provider_posting_id FROM jobs WHERE id = $1", first.ID).
			Scan(&title, &description, &url, &source, &posting)
		if err != nil {
			t.Fatal(err)
		}
		if title != wttj.Title || description != wttj.Description || url != libertyWTTJ || source != "wttj" || posting != "4LsTTZd3" {
			t.Errorf("job = %q %q %q %q, want the WTTJ version", title, url, source, posting)
		}
		if diff := cmp.Diff([]string{libertyWTTJ, libertyLinkedIn}, listingURLs(t, pool, first.ID)); diff != "" {
			t.Errorf("job_urls (-want +got):\n%s", diff)
		}
		var upgrades int
		err = pool.QueryRow(t.Context(), "SELECT count(*) FROM effect_outbox WHERE job_id = $1 AND NOT first_discovery", first.ID).Scan(&upgrades)
		if err != nil || upgrades != 1 {
			t.Errorf("non-first-discovery effects = %d, err = %v, want 1", upgrades, err)
		}
	})

	t.Run("the reverse order merges into the ATS Job and keeps its content", func(t *testing.T) {
		st, pool, _ := newUserStore(t)
		boardID, companyID := libertyBoard(t, st)
		wttj := atsListing(t, libertyWTTJ, boardID, companyID)
		first, _ := saveJob(t, st, wttj)

		got, status := saveJob(t, st, loadListing(t, libertyLinkedIn))
		if status != "merged" || got.ID != first.ID {
			t.Fatalf("status = %q, id = %q, want merged with id %q", status, got.ID, first.ID)
		}
		if n := jobCount(t, pool); n != 1 {
			t.Fatalf("jobs = %d, want 1", n)
		}
		var title, url string
		if err := pool.QueryRow(t.Context(), "SELECT title, url FROM jobs WHERE id = $1", first.ID).Scan(&title, &url); err != nil {
			t.Fatal(err)
		}
		if title != wttj.Title || url != libertyWTTJ {
			t.Errorf("job = %q %q, want the WTTJ version", title, url)
		}
	})

	t.Run("a different WTTJ role stays separate in both orders", func(t *testing.T) {
		for _, aggregatorFirst := range []bool{true, false} {
			st, pool, _ := newUserStore(t)
			boardID, companyID := libertyBoard(t, st)
			aggregator, other := loadListing(t, libertyLinkedIn), atsListing(t, libertyOtherRole, boardID, companyID)
			if aggregatorFirst {
				saveJob(t, st, aggregator)
				saveJob(t, st, other)
			} else {
				saveJob(t, st, other)
				saveJob(t, st, aggregator)
			}
			if n := jobCount(t, pool); n != 2 {
				t.Errorf("aggregatorFirst=%t: jobs = %d, want 2", aggregatorFirst, n)
			}
		}
	})

	t.Run("two ATS postings with an identical match key stay separate", func(t *testing.T) {
		st, pool, _ := newUserStore(t)
		boardID, companyID := libertyBoard(t, st)
		a := atsListing(t, libertyWTTJ, boardID, companyID)
		b := a
		b.URL, b.ProviderPostingID = "https://app.welcometothejungle.com/jobs/other", "other"
		saveJob(t, st, a)
		if _, status := saveJob(t, st, b); status != "new" {
			t.Errorf("status = %q, want new", status)
		}
		if n := jobCount(t, pool); n != 2 {
			t.Errorf("jobs = %d, want 2", n)
		}
	})

	t.Run("a board poll without the posting closes the upgraded Job", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		boardID, companyID := libertyBoard(t, st)
		trackCompany(t, st, userID, companyID, 60)
		if _, err := st.VerifyCompanyBoard(t.Context(), companyID, "wttj", "liberty-global", "user_confirmed", ""); err != nil {
			t.Fatal(err)
		}
		saveJob(t, st, loadListing(t, libertyLinkedIn))
		wttj := atsListing(t, libertyWTTJ, boardID, companyID)
		saveJob(t, st, wttj)

		env := boardEnv{st: st, pool: pool, board: dto.CompanyBoard{ID: boardID}, companyID: companyID}
		env.complete(t, env.claim(t, true), wttj)
		env.complete(t, env.claim(t, true))
		env.complete(t, env.claim(t, true))
		if !jobClosed(t, pool, libertyWTTJ) {
			t.Error("upgraded Job still open after the board dropped its posting")
		}
	})
}
