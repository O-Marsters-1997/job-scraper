package store_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

var pageNow = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func daysAgo(d float64) time.Time {
	return pageNow.Add(-time.Duration(d * float64(24*time.Hour)))
}

type listing struct {
	title       string
	slug        string
	source      string
	location    string
	arrangement string
	companyID   string
	posted      *time.Time
	discovered  time.Time
	score       *int
	band        string
	salaryMin   *float64
	salaryMax   *float64
	closed      bool
}

func insertListing(t *testing.T, pool *pgxpool.Pool, userID string, n int, l listing) string {
	t.Helper()
	id := fmt.Sprintf("30000000-0000-0000-0000-%012d", n)
	if l.title == "" {
		l.title = "Role"
	}
	if l.slug == "" {
		l.slug = "acme"
	}
	if l.source == "" {
		l.source = "greenhouse"
	}
	if l.discovered.IsZero() {
		l.discovered = daysAgo(0)
	}
	var companyID *string
	if l.companyID != "" {
		companyID = &l.companyID
	}
	_, err := pool.Exec(t.Context(),
		`INSERT INTO jobs (id,title,location,url,company_slug,source,posted_at,scraped_at,first_discovered_at,work_arrangement,salary_min,salary_max,closed_at,company_id)
		 VALUES ($1::uuid,$2,$3,'https://example.com/'||$1::text,$4,$5,$6,$7,$7,$8,$9,$10,CASE WHEN $11 THEN $7::timestamptz END,$12::uuid)`,
		id, l.title, l.location, l.slug, l.source, l.posted, l.discovered, l.arrangement, l.salaryMin, l.salaryMax, l.closed, companyID)
	if err != nil {
		t.Fatalf("insert listing %d: %v", n, err)
	}
	if l.score != nil {
		_, err := pool.Exec(t.Context(),
			`INSERT INTO job_scores (job_id,user_id,suitability_score,band,breakdown) VALUES ($1::uuid,$2::uuid,$3,NULLIF($4,''),'[]'::jsonb)`,
			id, userID, *l.score, l.band)
		if err != nil {
			t.Fatalf("score listing %d: %v", n, err)
		}
	}
	return id
}

func pageIDs(t *testing.T, st *store.Store, userID string, mutate func(*dto.JobPageOptions)) []string {
	t.Helper()
	page := pageOf(t, st, userID, mutate)
	return jobIDs(page.Items)
}

func pageOf(t *testing.T, st *store.Store, userID string, mutate func(*dto.JobPageOptions)) dto.JobPage {
	t.Helper()
	options := dto.JobPageOptions{Limit: 100, Now: pageNow, Sort: dto.JobSortRelevant}
	if mutate != nil {
		mutate(&options)
	}
	page, err := st.Page(t.Context(), userID, options)
	if err != nil {
		t.Fatalf("Page(%+v) err = %v", options, err)
	}
	return page
}

func wantIDs(t *testing.T, label string, want, got []string) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s ids (-want +got):\n%s", label, diff)
	}
}

func TestPage(t *testing.T) {
	t.Parallel()

	t.Run("hides closed, stale, blocked, graded-no and excluded jobs", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		other := pgtest.InsertUser(t, pool)
		visible := insertListing(t, pool, userID, 1, listing{score: new(50)})
		insertListing(t, pool, userID, 2, listing{score: new(50), closed: true})
		insertListing(t, pool, userID, 3, listing{score: new(50), discovered: daysAgo(91)})
		insertListing(t, pool, userID, 4, listing{score: new(50), slug: "globex"})
		blocked := insertListing(t, pool, userID, 5, listing{})
		dismissed := insertListing(t, pool, userID, 6, listing{score: new(50)})
		othersNo := insertListing(t, pool, userID, 7, listing{score: new(50), discovered: daysAgo(1)})
		if _, err := pool.Exec(t.Context(),
			`INSERT INTO job_scores (job_id,user_id,suitability_score,breakdown) VALUES ($1::uuid,$2::uuid,50,'[{"effect":"blocked"}]'::jsonb)`, blocked, userID); err != nil {
			t.Fatalf("block job: %v", err)
		}
		for _, g := range []struct{ user, job string }{{userID, dismissed}, {other, othersNo}} {
			if _, err := pool.Exec(t.Context(), `INSERT INTO job_grades (user_id,job_id,grade) VALUES ($1,$2,'no')`, g.user, g.job); err != nil {
				t.Fatalf("grade no: %v", err)
			}
		}

		got := pageIDs(t, st, userID, func(o *dto.JobPageOptions) { o.ExcludedCompanySlugs = []string{"globex"} })
		wantIDs(t, "Page()", []string{visible, othersNo}, got)
	})

	t.Run("windows on posted_at before first_discovered_at", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		fresh := insertListing(t, pool, userID, 1, listing{score: new(50), posted: new(daysAgo(89)), discovered: daysAgo(200)})
		insertListing(t, pool, userID, 2, listing{score: new(50), posted: new(daysAgo(91)), discovered: daysAgo(0)})

		wantIDs(t, "Page()", []string{fresh}, pageIDs(t, st, userID, nil))
	})

	t.Run("pages by offset and always reports the real total", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		var ids []string
		for n := 1; n <= 5; n++ {
			ids = append(ids, insertListing(t, pool, userID, n, listing{score: new(50), discovered: daysAgo(float64(n) / 10)}))
		}
		page := func(limit, offset int32) dto.JobPage {
			return pageOf(t, st, userID, func(o *dto.JobPageOptions) { o.Limit, o.Offset, o.Sort = limit, offset, dto.JobSortNewest })
		}

		for _, tt := range []struct {
			name       string
			limit, off int32
			want       []string
		}{
			{"first page", 2, 0, ids[0:2]},
			{"middle page", 2, 2, ids[2:4]},
			{"short last page", 2, 4, ids[4:5]},
			{"past the end", 2, 40, []string{}},
		} {
			got := page(tt.limit, tt.off)
			if diff := cmp.Diff(tt.want, jobIDs(got.Items)); diff != "" {
				t.Errorf("%s ids (-want +got):\n%s", tt.name, diff)
			}
			if got.Total != 5 {
				t.Errorf("%s total = %d, want 5", tt.name, got.Total)
			}
		}
	})

	t.Run("filters", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		ctx := t.Context()
		acme := insertCompany(t, pool, "acme")
		agency := insertCompany(t, pool, "recruiter")
		if err := st.SetCompanyKind(ctx, agency, dto.KindAgency); err != nil {
			t.Fatalf("SetCompanyKind() err = %v", err)
		}
		favCo := insertCompany(t, pool, "faved")
		if _, err := pool.Exec(ctx, `INSERT INTO company_favourites (user_id, company_id) VALUES ($1, $2)`, userID, favCo); err != nil {
			t.Fatalf("favourite company: %v", err)
		}

		unscored := insertListing(t, pool, userID, 1, listing{companyID: acme, title: "Gardener", slug: "acme", location: "Leeds", source: "linkedin"})
		low := insertListing(t, pool, userID, 2, listing{companyID: acme, score: new(30), arrangement: "Remote "})
		high := insertListing(t, pool, userID, 3, listing{companyID: acme, score: new(80), arrangement: "hybrid", source: "lever"})
		viaAgency := insertListing(t, pool, userID, 4, listing{companyID: agency, slug: "recruiter", score: new(60), arrangement: "onsite"})
		faved := insertListing(t, pool, userID, 5, listing{companyID: favCo, slug: "faved", score: new(60), arrangement: "mixed"})
		if _, err := pool.Exec(ctx, `INSERT INTO job_grades (user_id,job_id,grade) VALUES ($1,$2,'ok')`, userID, high); err != nil {
			t.Fatalf("grade: %v", err)
		}
		if err := st.MarkJobsSeen(ctx, userID, []string{low}, true); err != nil {
			t.Fatalf("MarkJobsSeen() err = %v", err)
		}
		all := []string{unscored, low, high, viaAgency, faved}

		for _, tt := range []struct {
			name   string
			mutate func(*dto.JobPageOptions)
			want   []string
		}{
			{"no filters", nil, all},
			{"company", func(o *dto.JobPageOptions) { o.CompanyID = acme }, []string{unscored, low, high}},
			{"scored only", func(o *dto.JobPageOptions) { o.ScoredOnly = true }, []string{low, high, viaAgency, faved}},
			{"min suitability", func(o *dto.JobPageOptions) { o.MinSuitability = new(int32(60)) }, []string{high, viaAgency, faved}},
			{"graded", func(o *dto.JobPageOptions) { o.Graded = "graded" }, []string{high}},
			{"ungraded", func(o *dto.JobPageOptions) { o.Graded = "ungraded" }, []string{unscored, low, viaAgency, faved}},
			{"seen", func(o *dto.JobPageOptions) { o.Seen = "seen" }, []string{low}},
			{"unseen", func(o *dto.JobPageOptions) { o.Seen = "unseen" }, []string{unscored, high, viaAgency, faved}},
			{"agency", func(o *dto.JobPageOptions) { o.Poster = dto.PosterAgency }, []string{viaAgency}},
			{"direct includes unclassified", func(o *dto.JobPageOptions) { o.Poster = dto.PosterDirect }, []string{unscored, low, high, faved}},
			{"favourite company", func(o *dto.JobPageOptions) { o.FavouriteOnly = true }, []string{faved}},
			{"one source", func(o *dto.JobPageOptions) { o.Sources = []string{"lever"} }, []string{high}},
			{"several sources", func(o *dto.JobPageOptions) { o.Sources = []string{"lever", "linkedin"} }, []string{unscored, high}},
			{"remote arrangement is trimmed and case-folded", func(o *dto.JobPageOptions) { o.Arrangements = []string{"remote"} }, []string{low}},
			{"several arrangements", func(o *dto.JobPageOptions) { o.Arrangements = []string{"hybrid", "onsite"} }, []string{high, viaAgency}},
			{"unrecognised arrangement is unknown", func(o *dto.JobPageOptions) { o.Arrangements = []string{"unknown"} }, []string{unscored, faved}},
			{"search title", func(o *dto.JobPageOptions) { o.Search = "garden" }, []string{unscored}},
			{"search location", func(o *dto.JobPageOptions) { o.Search = "LEEDS" }, []string{unscored}},
			{"search company slug", func(o *dto.JobPageOptions) { o.Search = "recruit" }, []string{viaAgency}},
			{"search source", func(o *dto.JobPageOptions) { o.Search = "lever" }, []string{high}},
			{"search treats wildcards literally", func(o *dto.JobPageOptions) { o.Search = "%" }, nil},
		} {
			got := pageOf(t, st, userID, func(o *dto.JobPageOptions) {
				o.Sort = dto.JobSortNewest
				if tt.mutate != nil {
					tt.mutate(o)
				}
			})
			if diff := cmp.Diff(sorted(tt.want), sorted(jobIDs(got.Items))); diff != "" {
				t.Errorf("%s ids (-want +got):\n%s", tt.name, diff)
			}
			if got.Total != len(tt.want) {
				t.Errorf("%s total = %d, want %d", tt.name, got.Total, len(tt.want))
			}
		}
	})

	t.Run("bands and scraped-since narrow the page", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		great := insertListing(t, pool, userID, 1, listing{score: new(90), band: "great", discovered: daysAgo(1)})
		good := insertListing(t, pool, userID, 2, listing{score: new(70), band: "good", discovered: daysAgo(2)})
		insertListing(t, pool, userID, 3, listing{score: new(40), band: "fair", discovered: daysAgo(1)})
		insertListing(t, pool, userID, 4, listing{score: new(95), band: "great", discovered: daysAgo(10)})

		got := pageIDs(t, st, userID, func(o *dto.JobPageOptions) {
			o.Sort = dto.JobSortBest
			o.Bands = []string{"great", "good"}
			o.ScrapedSince = new(daysAgo(7))
		})
		wantIDs(t, "Page()", []string{great, good}, got)
	})

	t.Run("salary filter keeps overlapping ranges and drops unparsed salaries", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		in := insertListing(t, pool, userID, 1, listing{salaryMin: new(80000.0), salaryMax: new(95000.0)})
		below := insertListing(t, pool, userID, 2, listing{salaryMin: new(30000.0), salaryMax: new(40000.0)})
		above := insertListing(t, pool, userID, 3, listing{salaryMin: new(150000.0), salaryMax: new(180000.0)})
		single := insertListing(t, pool, userID, 4, listing{salaryMin: new(90000.0)})
		none := insertListing(t, pool, userID, 5, listing{})

		for _, tt := range []struct {
			name     string
			min, max *float64
			want     []string
		}{
			{"any parsed salary", nil, nil, []string{in, below, above, single}},
			{"bounded", new(85000.0), new(100000.0), []string{in, single}},
			{"floor only", new(100000.0), nil, []string{above}},
			{"ceiling only", nil, new(35000.0), []string{below}},
		} {
			got := pageIDs(t, st, userID, func(o *dto.JobPageOptions) {
				o.Sort, o.SalaryOnly, o.SalaryMin, o.SalaryMax = dto.JobSortNewest, true, tt.min, tt.max
			})
			if diff := cmp.Diff(sorted(tt.want), sorted(got)); diff != "" {
				t.Errorf("%s ids (-want +got):\n%s", tt.name, diff)
			}
		}
		if got := pageIDs(t, st, userID, nil); len(got) != 5 || !slices.Contains(got, none) {
			t.Errorf("without the salary filter ids = %v, want all five including %s", got, none)
		}
	})

	t.Run("relevant sort decays score by age from posted_at, falling back to first_discovered_at", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		freshLow := insertListing(t, pool, userID, 1, listing{score: new(40), posted: new(daysAgo(0))})
		staleHigh := insertListing(t, pool, userID, 2, listing{score: new(90), posted: new(daysAgo(6))})
		unscoredFresh := insertListing(t, pool, userID, 3, listing{posted: new(daysAgo(0))})
		scoredOlder := insertListing(t, pool, userID, 4, listing{score: new(60), posted: new(daysAgo(3))})
		discoveredOnly := insertListing(t, pool, userID, 5, listing{score: new(80), discovered: daysAgo(6)})
		futurePosted := insertListing(t, pool, userID, 6, listing{score: new(45), posted: new(pageNow.Add(72 * time.Hour))})

		wantIDs(t, "relevant", []string{unscoredFresh, futurePosted, freshLow, scoredOlder, staleHigh, discoveredOnly}, pageIDs(t, st, userID, nil))
	})

	t.Run("relevant ties break on first_discovered_at then id", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		posted := new(daysAgo(1))
		olderDiscovery := insertListing(t, pool, userID, 3, listing{score: new(70), posted: posted, discovered: daysAgo(2)})
		newerDiscovery := insertListing(t, pool, userID, 2, listing{score: new(70), posted: posted, discovered: daysAgo(1)})
		sameA := insertListing(t, pool, userID, 4, listing{score: new(70), posted: posted, discovered: daysAgo(2)})

		wantIDs(t, "relevant", []string{newerDiscovery, olderDiscovery, sameA}, pageIDs(t, st, userID, nil))
	})

	t.Run("unknown sort behaves as relevant", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		fresh := insertListing(t, pool, userID, 1, listing{score: new(40), posted: new(daysAgo(0))})
		stale := insertListing(t, pool, userID, 2, listing{score: new(90), posted: new(daysAgo(6))})

		got := pageIDs(t, st, userID, func(o *dto.JobPageOptions) { o.Sort = "sideways" })
		wantIDs(t, "unknown sort", []string{fresh, stale}, got)
	})

	t.Run("newest sort orders by posted_at then first_discovered_at, ties on id", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		oldPost := insertListing(t, pool, userID, 1, listing{posted: new(daysAgo(5))})
		newPost := insertListing(t, pool, userID, 2, listing{posted: new(daysAgo(1))})
		discoveredMid := insertListing(t, pool, userID, 3, listing{discovered: daysAgo(3)})
		tieB := insertListing(t, pool, userID, 5, listing{posted: new(daysAgo(4))})
		tieA := insertListing(t, pool, userID, 4, listing{posted: new(daysAgo(4))})

		got := pageIDs(t, st, userID, func(o *dto.JobPageOptions) { o.Sort = dto.JobSortNewest })
		wantIDs(t, "newest", []string{newPost, discoveredMid, tieA, tieB, oldPost}, got)
	})

	t.Run("best sort orders by score with unscored last, then newest, then id", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		top := insertListing(t, pool, userID, 1, listing{score: new(90), posted: new(daysAgo(30))})
		tieNew := insertListing(t, pool, userID, 2, listing{score: new(70), posted: new(daysAgo(1))})
		tieOld := insertListing(t, pool, userID, 3, listing{score: new(70), posted: new(daysAgo(9))})
		unscoredNew := insertListing(t, pool, userID, 5, listing{posted: new(daysAgo(0))})
		unscoredTie := insertListing(t, pool, userID, 4, listing{posted: new(daysAgo(0))})

		got := pageIDs(t, st, userID, func(o *dto.JobPageOptions) { o.Sort = dto.JobSortBest })
		wantIDs(t, "best", []string{top, tieNew, tieOld, unscoredTie, unscoredNew}, got)
	})

	t.Run("rows carry the caller's grade, seen state and first discovered time, not the description", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		other := pgtest.InsertUser(t, pool)
		graded := insertListing(t, pool, userID, 1, listing{score: new(50), discovered: daysAgo(3)})
		ungraded := insertListing(t, pool, userID, 2, listing{score: new(50)})
		for _, g := range []struct{ user, job, grade string }{{userID, graded, "ok"}, {other, graded, "great"}, {other, ungraded, "ok"}} {
			if _, err := pool.Exec(t.Context(), `INSERT INTO job_grades (user_id,job_id,grade) VALUES ($1,$2,$3)`, g.user, g.job, g.grade); err != nil {
				t.Fatalf("grade: %v", err)
			}
		}
		for _, u := range []string{userID, other} {
			if err := st.MarkJobsSeen(t.Context(), u, []string{ungraded}, u == other); err != nil {
				t.Fatalf("MarkJobsSeen() err = %v", err)
			}
		}

		page := pageOf(t, st, userID, nil)
		if diff := cmp.Diff(map[string]string{graded: "ok", ungraded: ""}, jobGrades(page.Items)); diff != "" {
			t.Errorf("Page() grades (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(map[string]bool{graded: false, ungraded: false}, jobSeen(page.Items)); diff != "" {
			t.Errorf("Page() seen (-want +got):\n%s", diff)
		}
		for _, job := range page.Items {
			if job.Description != "" {
				t.Errorf("Page() job %s carried description %q, want it left to GetJob", job.ID, job.Description)
			}
			if job.ID == graded && !job.FirstDiscoveredAt.Equal(daysAgo(3)) {
				t.Errorf("Page() FirstDiscoveredAt = %v, want %v", job.FirstDiscoveredAt, daysAgo(3))
			}
		}
	})
}

func sorted(ids []string) []string {
	return slices.Sorted(slices.Values(ids))
}

func TestStore_JobStats(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	insertListing(t, pool, userID, 1, listing{score: new(50), source: "lever", discovered: daysAgo(0.1)})
	insertListing(t, pool, userID, 2, listing{score: new(50), source: "lever", discovered: daysAgo(0.2)})
	insertListing(t, pool, userID, 3, listing{score: new(50), source: "greenhouse", discovered: daysAgo(3)})
	insertListing(t, pool, userID, 4, listing{source: "linkedin"})
	insertListing(t, pool, userID, 5, listing{score: new(50), source: "ashby", closed: true})
	insertListing(t, pool, userID, 6, listing{score: new(50), source: "ashby", slug: "globex"})

	got, err := st.JobStats(t.Context(), userID, dto.JobStatsOptions{
		Now:                  pageNow,
		DayStart:             daysAgo(0.5),
		ExcludedCompanySlugs: []string{"globex"},
	})
	if err != nil {
		t.Fatalf("JobStats() err = %v", err)
	}
	want := dto.JobStats{Total: 3, Sources: 2, NewToday: 2, TodaySources: 1}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("JobStats() (-want +got):\n%s", diff)
	}
}
