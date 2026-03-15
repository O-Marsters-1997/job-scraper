package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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
	scraper := wis.New()

	jobs, err := scraper.FetchJobs(ctx)
	if err != nil {
		slog.Error("scrape error", slog.String("source", scraper.Name()), slog.Any("err", err))
	}

	slog.Info("scrape complete", slog.String("source", scraper.Name()), slog.Int("count", len(jobs)))

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
