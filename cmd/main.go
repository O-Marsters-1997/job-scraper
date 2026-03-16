package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := jobsdb.New(ctx, jobsdb.ConnString())
	if err != nil {
		slog.Error("db init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer db.Close()

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

	srcs := []sources.Source{wis.New()}
	if err := scraper.Run(ctx, srcs, db, q); err != nil {
		slog.Error("run failed", slog.Any("err", err))
		os.Exit(1)
	}
}
