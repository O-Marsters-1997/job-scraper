package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	kindCompany byte = iota + 1
	kindBoard
	kindJob
	kindUser
	kindRun
	kindRow
)

var (
	sources      = []string{"greenhouse", "ashby", "smartrecruiters", "lever", "pinpoint", "teamtailor", "recruitee", "personio"}
	sourceWeight = []float64{0.51, 0.21, 0.15, 0.06, 0.03, 0.02, 0.01, 0.01}
	levels       = []string{"Junior", "", "", "Senior", "Senior", "Staff", "Lead", "Principal"}
	roles        = []string{"Software Engineer", "Backend Engineer", "Frontend Engineer", "Full-stack Developer", "Platform Engineer", "Data Engineer", "Site Reliability Engineer", "Product Engineer", "Engineering Manager", "Mobile Engineer", "Machine Learning Engineer", "QA Engineer"}
	cities       = []string{"London", "Manchester", "Bristol", "Edinburgh", "Leeds", "Birmingham", "Cambridge", "Oxford", "Glasgow", "Remote (UK)", "Brighton", "Cardiff"}
	nameParts    = []string{"Acme", "Nimbus", "Lumen", "Kestrel", "Harbour", "Copper", "Juniper", "Vantage", "Orbit", "Meridian", "Beacon", "Fable", "Quarry", "Sable", "Tandem", "Willow", "Atlas", "Ember", "Pioneer", "Northwind"}
	nameSuffix   = []string{"Labs", "Analytics", "Health", "Systems", "Payments", "Robotics", "Studio", "Logistics", "Energy", "Software", "Group", "Technologies"}
	arrangements = []string{"onsite", "hybrid", "hybrid", "remote", ""}
	companyKinds = []string{"unclassified", "unclassified", "likely_direct", "direct", "likely_agency", "agency"}
	eventTypes   = []string{"job_opened", "job_opened", "job_opened", "job_dismissed", "application_created", "application_status_changed", "alert_opened"}
	criteria     = []string{"role:backend", "role:frontend", "role:full-stack", "work:hybrid", "work:remote", "tech:react", "tech:node-js", "tech:next-js", "tech:postgres", "tech:graphql", "tech:aws", "tech:docker", "tech:llms", "tech:typescript", "tech:php", "tech:laravel", "tech:spring", "tech:java", "domain:devtools", "domain:ai-tooling", "domain:climate", "domain:health", "domain:gambling", "domain:crypto", "domain:adtech", "stage:seed", "stage:series_a", "stage:series_b", "size:startup", "size:scaleup", "tech:azure", "tech:angular", "tech:go", "tech:mongodb", "domain:ecommerce", "domain:legal", "domain:edtech", "domain:logistics", "salary", "seniority", "seniority:mid"}
	effects      = []string{"meets", "misses", "unknown", "unknown", "neutral"}
)

type userPlan struct {
	tier                                                              string
	n                                                                 int
	tracked, scores, rejections, views, grades, apps, tasks, comments int
	notes, favourites, events                                         int
}

var userPlans = []userPlan{
	{"power", 1, 1500, 5000, 75000, 3000, 500, 200, 300, 400, 200, 100, 5000},
	{"typical", 3, 50, 300, 5000, 150, 30, 10, 10, 10, 10, 10, 200},
	{"light", 16, 5, 30, 500, 15, 3, 1, 1, 1, 1, 1, 20},
}

type seeder struct {
	rng       *rand.Rand
	seed      uint64
	scale     float64
	now       time.Time
	corpus    string
	breakdown [][]byte

	nCompanies, nJobs, nBoards int
	companyFirstBoard          []int32
	companyBoardCount          []uint8
	companyJobs                []int32
	boardCompany               []int32
	boardSource                []uint8
	boardVerified              []bool
	verifiedBoards             []int32
	jobCompany                 []int32
	jobClosed                  []bool
	jobDescLen                 []int32
	rowSeq                     uint64
}

func newSeeder(seed uint64, scale float64) *seeder {
	s := &seeder{
		rng:   rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		seed:  seed,
		scale: scale,
		now:   time.Now().UTC().Truncate(24 * time.Hour),
	}
	s.corpus = s.buildCorpus(64 << 10)
	for range 64 {
		s.breakdown = append(s.breakdown, s.buildBreakdown(false))
	}
	for range 4 {
		s.breakdown = append(s.breakdown, s.buildBreakdown(true))
	}
	return s
}

func (s *seeder) n(base int) int { return max(1, int(math.Round(float64(base)*s.scale))) }

func (s *seeder) id(kind byte, i int) [16]byte {
	var b [16]byte
	b[0] = kind
	binary.BigEndian.PutUint32(b[1:5], uint32(s.seed))
	b[6] = 0x40
	binary.BigEndian.PutUint64(b[8:16], uint64(i))
	b[8] = 0x80
	return b
}

func (s *seeder) nextRowID() [16]byte {
	s.rowSeq++
	return s.id(kindRow, int(s.rowSeq))
}

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

func (s *seeder) weighted(weights []float64) int {
	u := s.rng.Float64()
	for i, w := range weights {
		if u < w {
			return i
		}
		u -= w
	}
	return len(weights) - 1
}

func (s *seeder) pastWithin(d time.Duration) time.Time {
	return s.now.Add(-time.Duration(s.rng.Int64N(int64(d))))
}

func (s *seeder) buildCorpus(size int) string {
	words := strings.Fields("we are looking for an engineer to join our team and help build reliable scalable services " +
		"you will work closely with product design and data colleagues shipping features to customers every week " +
		"experience with go postgres typescript react kubernetes aws observability testing and continuous delivery " +
		"responsibilities include designing apis owning services on call mentoring reviewing code improving performance " +
		"benefits pension private healthcare flexible working learning budget share options generous holiday")
	var b strings.Builder
	for b.Len() < size {
		for range 12 + s.rng.IntN(20) {
			b.WriteString(pick(s.rng, words))
			b.WriteByte(' ')
		}
		b.WriteString(".\n\n")
	}
	return b.String()
}

func (s *seeder) text(n int) string {
	n = min(n, len(s.corpus)-1)
	off := s.rng.IntN(len(s.corpus) - n)
	return s.corpus[off : off+n]
}

type criterion struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Effect     string `json:"effect"`
	Stance     string `json:"stance"`
	Resolved   string `json:"resolved"`
	Corrected  bool   `json:"corrected"`
	Overridden bool   `json:"overridden"`
}

func (s *seeder) buildBreakdown(blocked bool) []byte {
	out := make([]criterion, 0, len(criteria))
	for _, k := range criteria {
		e := pick(s.rng, effects)
		resolved := map[string]string{"meets": "yes", "misses": "no", "neutral": "no"}[e]
		if resolved == "" {
			resolved = "unknown"
		}
		out = append(out, criterion{Key: k, Label: k, Effect: e, Stance: pick(s.rng, []string{"nice", "nice", "avoid"}), Resolved: resolved})
	}
	if blocked {
		out[s.rng.IntN(len(out))].Effect = "blocked"
	}
	b, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}
	return b
}

func (s *seeder) buildCatalogue() {
	s.nCompanies, s.nJobs = s.n(25000), s.n(80000)
	s.companyFirstBoard = make([]int32, s.nCompanies)
	s.companyBoardCount = make([]uint8, s.nCompanies)
	s.companyJobs = make([]int32, s.nCompanies)
	for c := range s.nCompanies {
		k := []uint8{1, 1, 1, 1, 1, 1, 1, 2, 2, 3}[s.rng.IntN(10)]
		if s.rng.IntN(10) == 0 {
			k = 0
		}
		s.companyFirstBoard[c] = int32(len(s.boardCompany))
		s.companyBoardCount[c] = k
		for range k {
			b := int32(len(s.boardCompany))
			s.boardCompany = append(s.boardCompany, int32(c))
			s.boardSource = append(s.boardSource, uint8(s.weighted(sourceWeight)))
			verified := s.rng.IntN(100) < 63
			s.boardVerified = append(s.boardVerified, verified)
			if verified {
				s.verifiedBoards = append(s.verifiedBoards, b)
			}
		}
	}
	s.nBoards = len(s.boardCompany)

	s.jobCompany = make([]int32, s.nJobs)
	s.jobClosed = make([]bool, s.nJobs)
	s.jobDescLen = make([]int32, s.nJobs)
	for j := range s.nJobs {
		u := s.rng.Float64()
		c := int32(float64(s.nCompanies) * u * u * u)
		s.jobCompany[j] = c
		s.companyJobs[c]++
		s.jobClosed[j] = s.rng.IntN(10) == 0
		s.jobDescLen[j] = int32(min(30000, max(400, math.Exp(s.rng.NormFloat64()*0.6+8.7))))
	}
}

func (s *seeder) copy(ctx context.Context, tx pgx.Tx, table string, cols []string, n int, row func(i int) []any) error {
	start := time.Now()
	got, err := tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromSlice(n, func(i int) ([]any, error) {
		if i > 0 && i%200000 == 0 {
			fmt.Printf("  %s: %d/%d\n", table, i, n)
		}
		return row(i), nil
	}))
	if err != nil {
		return fmt.Errorf("copy %s: %w", table, err)
	}
	if got >= 10000 {
		fmt.Printf("%-24s %9d rows  %6.1fs\n", table, got, time.Since(start).Seconds())
	}
	return nil
}

func (s *seeder) companyName(c int) string {
	return fmt.Sprintf("%s %s %d", nameParts[c%len(nameParts)], nameSuffix[(c/len(nameParts))%len(nameSuffix)], c)
}

func (s *seeder) jobURL(j int) string { return fmt.Sprintf("https://perf.example/jobs/%d", j) }

func (s *seeder) seedCatalogue(ctx context.Context, tx pgx.Tx) error {
	err := s.copy(ctx, tx, "companies", []string{"id", "slug", "name", "kind", "domain", "first_seen_at", "last_crawled_at"}, s.nCompanies, func(c int) []any {
		seen := s.pastWithin(365 * 24 * time.Hour)
		return []any{s.id(kindCompany, c), fmt.Sprintf("perf-%d", c), s.companyName(c), pick(s.rng, companyKinds),
			fmt.Sprintf("perf-%d.example", c), seen, seen.Add(24 * time.Hour)}
	})
	if err != nil {
		return err
	}
	err = s.copy(ctx, tx, "company_boards", []string{"id", "company_id", "source", "board_token", "status", "verification_method", "verified_at", "created_at"}, s.nBoards, func(b int) []any {
		var status, method any = "candidate", nil
		var verifiedAt any
		if s.boardVerified[b] {
			status, method, verifiedAt = "verified", "discovered", s.pastWithin(300*24*time.Hour)
		}
		return []any{s.id(kindBoard, b), s.id(kindCompany, int(s.boardCompany[b])), sources[s.boardSource[b]],
			fmt.Sprintf("perf-%d", b), status, method, verifiedAt, s.pastWithin(365 * 24 * time.Hour)}
	})
	if err != nil {
		return err
	}
	err = s.copy(ctx, tx, "board_poll_state", []string{"board_id", "last_completed_at", "last_started_at", "next_due_at"}, len(s.verifiedBoards), func(i int) []any {
		done := s.pastWithin(6 * time.Hour)
		return []any{s.id(kindBoard, int(s.verifiedBoards[i])), done, done.Add(-30 * time.Second), done.Add(6 * time.Hour)}
	})
	if err != nil {
		return err
	}
	err = s.copy(ctx, tx, "jobs", []string{"id", "title", "location", "url", "company_slug", "source", "posted_at", "scraped_at", "first_discovered_at",
		"description", "salary_raw", "salary_min", "salary_max", "work_arrangement", "closed_at", "company_id", "primary_board_id",
		"content_fingerprint", "match_title", "match_location"}, s.nJobs, func(j int) []any {
		c := int(s.jobCompany[j])
		title := strings.TrimSpace(pick(s.rng, levels) + " " + pick(s.rng, roles))
		loc := pick(s.rng, cities)
		posted := s.pastWithin(365 * 24 * time.Hour)
		source, board := "linkedin", any(nil)
		if s.companyBoardCount[c] > 0 {
			fb := int(s.companyFirstBoard[c])
			source, board = sources[s.boardSource[fb]], s.id(kindBoard, fb)
		}
		var closed any
		if s.jobClosed[j] {
			closed = posted.Add(time.Duration(1+s.rng.IntN(60)) * 24 * time.Hour)
		}
		var salRaw, salMin, salMax any = "", nil, nil
		if s.rng.IntN(3) == 0 {
			lo := float64(40+s.rng.IntN(80)) * 1000
			salRaw, salMin, salMax = fmt.Sprintf("£%.0fk - £%.0fk", lo/1000, lo/1000+15), lo, lo+15000
		}
		return []any{s.id(kindJob, j), title, loc, s.jobURL(j), fmt.Sprintf("perf-%d", c), source, posted, posted, posted,
			s.text(int(s.jobDescLen[j])), salRaw, salMin, salMax, pick(s.rng, arrangements), closed, s.id(kindCompany, c), board,
			fmt.Sprintf("%016x", s.rng.Uint64()), strings.ToLower(title), strings.ToLower(loc)}
	})
	if err != nil {
		return err
	}
	err = s.copy(ctx, tx, "job_urls", []string{"job_id", "normalized_url", "source", "first_seen_at"}, s.nJobs, func(j int) []any {
		return []any{s.id(kindJob, j), strings.TrimPrefix(s.jobURL(j), "https://"), "perf", s.pastWithin(365 * 24 * time.Hour)}
	})
	if err != nil {
		return err
	}
	var observed []int32
	for j := range s.nJobs {
		if s.companyBoardCount[s.jobCompany[j]] > 0 {
			observed = append(observed, int32(j))
		}
	}
	err = s.copy(ctx, tx, "board_job_observations", []string{"board_id", "job_id", "last_snapshot_version", "last_seen_at"}, len(observed), func(i int) []any {
		j := int(observed[i])
		return []any{s.id(kindBoard, int(s.companyFirstBoard[s.jobCompany[j]])), s.id(kindJob, j), int64(1), s.pastWithin(24 * time.Hour)}
	})
	if err != nil {
		return err
	}
	return s.seedScrapeRuns(ctx, tx)
}

func (s *seeder) seedScrapeRuns(ctx context.Context, tx pgx.Tx) error {
	nRuns := s.n(3000)
	err := s.copy(ctx, tx, "scrape_runs", []string{"id", "kind", "source", "board_id", "trigger", "started_at", "finished_at", "stop_reason", "pages", "listings", "new_listings"}, nRuns, func(r int) []any {
		b := int(pick(s.rng, s.verifiedBoards))
		start := s.pastWithin(365 * 24 * time.Hour)
		return []any{s.id(kindRun, r), "board", sources[s.boardSource[b]], s.id(kindBoard, b), pick(s.rng, []string{"scheduled", "scheduled", "manual"}),
			start, start.Add(time.Duration(2+s.rng.IntN(60)) * time.Second), "last_page", 1 + s.rng.IntN(5), s.rng.IntN(300), s.rng.IntN(20)}
	})
	if err != nil {
		return err
	}
	type change struct{ run, job int }
	var changes []change
	for r := range nRuns {
		k := 1 + s.rng.IntN(19)
		if r == 0 {
			k = 500
		}
		seen := map[int]bool{}
		for range k {
			j := s.rng.IntN(s.nJobs)
			if !seen[j] {
				seen[j] = true
				changes = append(changes, change{r, j})
			}
		}
	}
	return s.copy(ctx, tx, "scrape_run_job_changes", []string{"run_id", "job_id", "change", "snapshot"}, len(changes), func(i int) []any {
		c := changes[i]
		snap := fmt.Appendf(nil, `{"title":%q,"url":%q}`, pick(s.rng, roles), s.jobURL(c.job))
		return []any{s.id(kindRun, c.run), s.id(kindJob, c.job), pick(s.rng, []string{"new", "new", "removed", "reopened"}), snap}
	})
}

type seededUser struct {
	Tier     string `json:"tier"`
	Username string `json:"username"`
	ID       string `json:"id"`
	Password string `json:"password"`
	Session  string `json:"session_cookie"`

	idx                         int
	hotJob                      int
	firstScored                 int
	statuses, taskTypes         [][16]byte
	firstApplication, firstTask [16]byte
}

func (s *seeder) seedUserRows(ctx context.Context, tx pgx.Tx, u *seededUser, p userPlan) error {
	uid := s.id(kindUser, u.idx)
	perm := s.rng.Perm(s.nJobs)
	nScores := min(s.n(p.scores), s.nJobs)
	nRej := min(s.n(p.rejections), s.nJobs-nScores)
	scored, rejected := perm[:nScores], perm[nScores:nScores+nRej]
	u.firstScored, u.hotJob = scored[0], scored[len(scored)-1]
	subset := func(n int) []int { return scored[:min(s.n(n), len(scored))] }

	err := s.copy(ctx, tx, "job_scores", []string{"job_id", "user_id", "suitability_score", "band", "breakdown", "score_model", "score_fingerprint", "cost"}, len(scored), func(i int) []any {
		score := int32(min(100, max(0, int(s.rng.NormFloat64()*18+50))))
		band := "poor"
		switch {
		case score >= 80:
			band = "great"
		case score >= 65:
			band = "good"
		case score >= 50:
			band = "fair"
		}
		return []any{s.id(kindJob, scored[i]), uid, score, band, pick(s.rng, s.breakdown), "perf-seed", fmt.Sprintf("%016x", s.rng.Uint64()), 0.001}
	})
	if err != nil {
		return err
	}
	err = s.copy(ctx, tx, "job_rejections", []string{"user_id", "job_id", "kind", "reason", "content_fingerprint"}, len(rejected), func(i int) []any {
		return []any{uid, s.id(kindJob, rejected[i]), pick(s.rng, []string{"filter", "filter", "age"}), "title excluded", "perf"}
	})
	if err != nil {
		return err
	}

	var tracked []int
	isTracked := map[int]bool{}
	for _, i := range s.rng.Perm(len(s.verifiedBoards))[:min(s.n(p.tracked), len(s.verifiedBoards))] {
		c := int(s.boardCompany[s.verifiedBoards[i]])
		if !isTracked[c] {
			isTracked[c] = true
			tracked = append(tracked, c)
		}
	}
	err = s.copy(ctx, tx, "tracked_companies", []string{"user_id", "company_id", "enabled", "check_interval_minutes", "review_state"}, len(tracked), func(i int) []any {
		return []any{uid, s.id(kindCompany, tracked[i]), s.rng.IntN(10) > 0, int32(360), pick(s.rng, []string{"kept", "kept", "new", "dismissed"})}
	})
	if err != nil {
		return err
	}
	favs := tracked[:min(s.n(p.favourites), len(tracked))]
	err = s.copy(ctx, tx, "company_favourites", []string{"user_id", "company_id"}, len(favs), func(i int) []any {
		return []any{uid, s.id(kindCompany, favs[i])}
	})
	if err != nil {
		return err
	}

	views := subset(p.views)
	err = s.copy(ctx, tx, "job_views", []string{"user_id", "job_id", "seen_at"}, len(views), func(i int) []any {
		return []any{uid, s.id(kindJob, views[i]), s.pastWithin(365 * 24 * time.Hour)}
	})
	if err != nil {
		return err
	}
	grades := subset(p.grades)
	err = s.copy(ctx, tx, "job_grades", []string{"user_id", "job_id", "grade", "reasons"}, len(grades), func(i int) []any {
		return []any{uid, s.id(kindJob, grades[i]), pick(s.rng, []string{"great", "ok", "no"}), []string{}}
	})
	if err != nil {
		return err
	}
	apps := subset(p.apps)
	u.firstApplication = s.nextRowID()
	err = s.copy(ctx, tx, "applications", []string{"id", "user_id", "job_id", "status_id", "applied_at", "salary_info", "chase_by"}, len(apps), func(i int) []any {
		id := u.firstApplication
		if i > 0 {
			id = s.nextRowID()
		}
		applied := s.pastWithin(365 * 24 * time.Hour)
		return []any{id, uid, s.id(kindJob, apps[i]), pick(s.rng, u.statuses), applied, "", applied.Add(14 * 24 * time.Hour)}
	})
	if err != nil {
		return err
	}
	tasks := subset(p.tasks)
	u.firstTask = s.nextRowID()
	err = s.copy(ctx, tx, "tasks", []string{"id", "user_id", "type_id", "job_id", "link", "notes", "completed_at"}, len(tasks), func(i int) []any {
		id := u.firstTask
		if i > 0 {
			id = s.nextRowID()
		}
		var done any
		if s.rng.IntN(2) == 0 {
			done = s.pastWithin(180 * 24 * time.Hour)
		}
		return []any{id, uid, pick(s.rng, u.taskTypes), s.id(kindJob, tasks[i]), s.jobURL(tasks[i]), s.text(80 + s.rng.IntN(400)), done}
	})
	if err != nil {
		return err
	}
	commentJob := func(i int) int {
		if i%10 == 0 {
			return u.hotJob
		}
		return scored[i%len(scored)]
	}
	err = s.copy(ctx, tx, "job_comments", []string{"id", "user_id", "job_id", "body", "quote"}, s.n(p.comments), func(i int) []any {
		return []any{s.nextRowID(), uid, s.id(kindJob, commentJob(i)), s.text(40 + s.rng.IntN(300)), s.text(20 + s.rng.IntN(120))}
	})
	if err != nil {
		return err
	}
	notes := subset(p.notes)
	err = s.copy(ctx, tx, "job_notes", []string{"id", "user_id", "job_id", "body"}, len(notes), func(i int) []any {
		return []any{s.nextRowID(), uid, s.id(kindJob, notes[i]), s.text(100 + s.rng.IntN(1500))}
	})
	if err != nil {
		return err
	}
	return s.copy(ctx, tx, "events", []string{"user_id", "type", "subject_id", "props", "created_at"}, s.n(p.events), func(i int) []any {
		return []any{uid, pick(s.rng, eventTypes), s.id(kindJob, scored[i%len(scored)]), []byte("{}"), s.pastWithin(365 * 24 * time.Hour)}
	})
}
