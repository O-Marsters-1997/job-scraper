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
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/worker"
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

	cr, err := schedule.New(srcs).Initialize(ctx, db, q)
	if err != nil {
		slog.Error("scheduler init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer cr.Stop()

	scraper.Seed(ctx, srcs, db, q)

	slog.Info("queue processing worker starting")
	if err := worker.Run(ctx, q, func(ctx context.Context, url string) error {
		job, err := sources.Dispatch(ctx, srcs, url)
		if err != nil {
			slog.Error("dispatch failed", slog.String("url", url), slog.Any("err", err))
			return err
		}
		if err := db.UpsertJob(ctx, job); err != nil {
			slog.Error("upsert failed", slog.String("url", url), slog.Any("err", err))
			return err
		}
		slog.Info("job upserted", slog.String("url", url), slog.String("title", job.Title))
		return nil
	}); err != nil {
		slog.Error("worker failed", slog.Any("err", err))
	}
}
