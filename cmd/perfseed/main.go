package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
)

const manifestPath = ".perf-reports/seed-manifest.json"

var countedTables = []string{"companies", "company_boards", "board_poll_state", "jobs", "job_urls", "board_job_observations",
	"scrape_runs", "scrape_run_job_changes", "users", "job_scores", "job_rejections", "tracked_companies", "company_favourites",
	"job_views", "job_grades", "applications", "tasks", "job_comments", "job_notes", "events"}

type ref struct {
	Typical string `json:"typical"`
	Heavy   string `json:"heavy"`
}

type manifest struct {
	Scale     float64          `json:"scale"`
	Seed      uint64           `json:"seed"`
	SeededAt  time.Time        `json:"seeded_at"`
	RowCounts map[string]int64 `json:"row_counts"`
	Users     []seededUser     `json:"users"`
	KnownIDs  map[string]ref   `json:"known_ids"`
}

func main() {
	slog.SetDefault(logger.MustFromEnv())
	scaleFlag := flag.String("scale", "1x", "multiplier on the 1x row counts: 1x, 10x or a number")
	seed := flag.Uint64("seed", 42, "random seed; the same seed and scale give the same data")
	confirmed := flag.Bool("yes-wipe-dev-data", false, "confirm wiping jobs, companies, scores and perf-* users in the local dev database")
	flag.Parse()

	if err := run(context.Background(), *scaleFlag, *seed, *confirmed); err != nil {
		fmt.Fprintln(os.Stderr, "perfseed:", err)
		os.Exit(1)
	}
}

func parseScale(v string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSuffix(strings.ToLower(v), "x"), 64)
	if err != nil || f <= 0 {
		return 0, fmt.Errorf("invalid --scale %q", v)
	}
	return f, nil
}

func run(ctx context.Context, scaleFlag string, seed uint64, confirmed bool) error {
	scale, err := parseScale(scaleFlag)
	if err != nil {
		return err
	}
	connStr, err := db.ConnString()
	if err != nil {
		return err
	}
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parse postgres config: %w", err)
	}
	if err := checkGuard(os.Getenv("ALLOY_ENV"), cfg.ConnConfig.Host, confirmed); err != nil {
		return fmt.Errorf("refusing to seed, nothing was touched:\n%w", err)
	}

	start := time.Now()
	pool, err := db.Connect(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	s := newSeeder(seed, scale)
	s.buildCatalogue()
	users, err := s.seedAll(ctx, pool)
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, "ANALYZE"); err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	counts := map[string]int64{}
	for _, t := range countedTables {
		var n int64
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{t}.Sanitize()).Scan(&n); err != nil {
			return fmt.Errorf("count %s: %w", t, err)
		}
		counts[t] = n
	}
	m := manifest{Scale: scale, Seed: seed, SeededAt: time.Now().UTC(), RowCounts: counts, Users: users, KnownIDs: s.knownIDs(users)}
	if err := writeManifest(m); err != nil {
		return err
	}

	fmt.Printf("\nseeded scale=%gx seed=%d in %.1fs; manifest at %s\n", scale, seed, time.Since(start).Seconds(), manifestPath)
	for _, t := range countedTables {
		fmt.Printf("  %-24s %9d\n", t, counts[t])
	}
	return nil
}

func (s *seeder) seedAll(ctx context.Context, pool *pgxpool.Pool) ([]seededUser, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.WarnContext(ctx, "rollback failed", slog.Any("err", err))
		}
	}()

	eventType, err := tx.Conn().LoadType(ctx, "event_type")
	if err != nil {
		return nil, fmt.Errorf("load event_type: %w", err)
	}
	tx.Conn().TypeMap().RegisterType(eventType)

	if err := wipe(ctx, tx); err != nil {
		return nil, err
	}
	if err := s.seedCatalogue(ctx, tx); err != nil {
		return nil, err
	}

	taskModule := tasks.NewFacade(pool)
	apps := applications.New(pool, events.New(pool, scoring.NewFacade(pool)), taskModule)
	var users []seededUser
	for _, p := range userPlans {
		for k := range p.n {
			u := seededUser{Tier: p.tier, Username: fmt.Sprintf("perf-%s-%d", p.tier, k+1), idx: len(users)}
			if err := s.createUser(ctx, tx, &u, apps, taskModule); err != nil {
				return nil, err
			}
			if err := s.seedUserRows(ctx, tx, &u, p); err != nil {
				return nil, err
			}
			users = append(users, u)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return users, nil
}

type defaultsSeeder interface {
	SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *seeder) createUser(ctx context.Context, tx pgx.Tx, u *seededUser, seeders ...defaultsSeeder) error {
	id := s.id(kindUser, u.idx)
	u.ID = uuid.UUID(id).String()
	password, err := randomToken()
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	u.Password = password
	if _, err := tx.Exec(ctx, "INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)", id, u.Username, string(hash)); err != nil {
		return fmt.Errorf("insert user %s: %w", u.Username, err)
	}
	for _, sd := range seeders {
		if err := sd.SeedDefaults(ctx, tx, u.ID); err != nil {
			return fmt.Errorf("seed defaults for %s: %w", u.Username, err)
		}
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(token))
	if _, err := tx.Exec(ctx, "INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", id, sum[:], time.Now().Add(30*24*time.Hour)); err != nil {
		return fmt.Errorf("insert session for %s: %w", u.Username, err)
	}
	u.Session = token

	if u.statuses, err = ids(ctx, tx, "SELECT id FROM application_statuses WHERE user_id = $1 ORDER BY created_at, id", id); err != nil {
		return err
	}
	if u.taskTypes, err = ids(ctx, tx, "SELECT id FROM task_types WHERE user_id = $1 ORDER BY created_at, id", id); err != nil {
		return err
	}
	if len(u.statuses) == 0 || len(u.taskTypes) == 0 {
		return fmt.Errorf("user %s has no default statuses or task types after seeding defaults", u.Username)
	}
	return nil
}

func ids(ctx context.Context, tx pgx.Tx, sql string, arg any) ([][16]byte, error) {
	rows, err := tx.Query(ctx, sql, arg)
	if err != nil {
		return nil, fmt.Errorf("query ids: %w", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[[16]byte])
	if err != nil {
		return nil, fmt.Errorf("collect ids: %w", err)
	}
	return out, nil
}

func (s *seeder) knownIDs(users []seededUser) map[string]ref {
	str := func(b [16]byte) string { return uuid.UUID(b).String() }
	typicalCompany := 0
	for c := s.nCompanies - 1; c >= 0; c-- {
		if s.companyJobs[c] >= 15 && s.companyJobs[c] <= 40 && s.companyBoardCount[c] > 0 {
			typicalCompany = c
			break
		}
	}
	power, typical := users[0], users[1]
	return map[string]ref{
		"user":        {Typical: typical.ID, Heavy: power.ID},
		"company":     {Typical: str(s.id(kindCompany, typicalCompany)), Heavy: str(s.id(kindCompany, 0))},
		"job":         {Typical: str(s.id(kindJob, power.firstScored)), Heavy: str(s.id(kindJob, power.hotJob))},
		"application": {Typical: str(typical.firstApplication), Heavy: str(power.firstApplication)},
		"task":        {Typical: str(typical.firstTask), Heavy: str(power.firstTask)},
		"scrape_run":  {Typical: str(s.id(kindRun, 1)), Heavy: str(s.id(kindRun, 0))},
	}
}

func writeManifest(m manifest) error {
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, b, 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
