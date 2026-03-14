package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/utils"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// ── Postgres ─────────────────────────────────────────────────────────────
	connString := postgresConnString()

	db, err := db.New(ctx, connString)
	if err != nil {
		slog.Error("db init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer db.Close()

	// ── Valkey ────────────────────────────────────────────────────────────────
	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}

	q, err := queue.New(valkeyAddr)
	if err != nil {
		slog.Error("queue init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer q.Close()

	// ── Scrape ───────────────────────────────────────────────────────────────
	scraper, err := greenhouse.New(greenhouse.Config{
		BoardTokens: []string{"greenhouse"},
	})
	if err != nil {
		slog.Error("scraper init failed", slog.Any("err", err))
		os.Exit(1)
	}

	jobs, err := scraper.FetchJobs(ctx)
	if err != nil {
		slog.Error("scrape error", slog.String("source", scraper.Name()), slog.Any("err", err))
	}

	slog.Info("scrape complete", slog.String("source", scraper.Name()), slog.Int("count", len(jobs)))

	// ── Persist ──────────────────────────────────────────────────────────────
	if err := db.UpsertJobs(ctx, jobs); err != nil {
		slog.Error("upsert jobs failed", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("jobs persisted", slog.Int("count", len(jobs)))

	// ── Deduplicate ──────────────────────────────────────────────────────────
	seen := make(map[string]struct{}, len(jobs))
	var urls []string
	for _, j := range jobs {
		if j.URL == "" {
			continue
		}
		if _, ok := seen[j.URL]; ok {
			continue
		}
		seen[j.URL] = struct{}{}
		urls = append(urls, j.URL)
	}

	slog.Info("deduplication complete", slog.Int("unique", len(urls)))

	// ── Enqueue ──────────────────────────────────────────────────────────────
	if err := q.Enqueue(ctx, urls, time.Now()); err != nil {
		slog.Error("enqueue failed", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("enqueue complete", slog.Int("count", len(urls)))
}

func postgresConnString() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	host := utils.MustGetEnv("POSTGRES_HOST")
	port := utils.MustGetEnv("POSTGRES_PORT")
	user := utils.MustGetEnv("POSTGRES_USER")
	pass := utils.MustGetEnv("POSTGRES_PASSWORD")
	name := utils.MustGetEnv("POSTGRES_DB")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)
}
