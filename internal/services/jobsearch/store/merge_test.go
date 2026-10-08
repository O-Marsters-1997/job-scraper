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
)

const (
	heidiLinkedIn = "https://www.linkedin.com/jobs/view/4474766548"
	heidiWIS      = "https://workinstartups.com/details/5909695355"
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
