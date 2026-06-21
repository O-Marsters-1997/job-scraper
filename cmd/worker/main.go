package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"

	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/worker"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	connStr, err := jobsdb.ConnString()
	if err != nil {
		slog.Error("db config invalid", slog.Any("err", err))
		os.Exit(1)
	}
	db, err := jobsdb.New(ctx, connStr)
	if err != nil {
		slog.Error("db init failed",
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	defer db.Close()

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}

	q, err := queue.New(valkeyAddr)
	if err != nil {
		slog.Error("queue init failed",
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	slog.Info("queue client ready", slog.String("addr", valkeyAddr))
	defer q.Close()

	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		slog.Error("API_BASE_URL is required")
		os.Exit(1)
	}
	ingestToken := os.Getenv("INGEST_SERVICE_TOKEN")
	exporter := scraper.NewAPIExporter(apiBaseURL, ingestToken)

	scoringUserID := os.Getenv("SCORING_USER_ID")

	targets, err := db.ListEnabledSourceTargets(ctx)
	if err != nil {
		slog.Error("load source targets failed", slog.Any("err", err))
		os.Exit(1)
	}
	srcs := builder.BuildSources(targets)
	slog.Info("sources built from db", slog.Int("count", len(srcs)))

	orch := scraper.New(srcs, db, q)
	orch.WithExporter(exporter)

	if scoringUserID != "" {
		orch.WithRelevanceGate(score.NewHeuristicScorer(), db, db, scoringUserID)
		slog.Info("relevance gate enabled", slog.String("user_id", scoringUserID))
	}

	if err := orch.Start(ctx); err != nil {
		slog.Error("orchestrator start failed",
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	defer orch.Stop()

	cr := cron.New()

	if _, err := cr.AddFunc("@daily", func() {
		if err := db.DeleteExpiredSessions(ctx); err != nil {
			slog.Error("session cleanup failed",
				slog.Any("err", err),
			)
		}
	}); err != nil {
		slog.Error("session cleanup cron schedule failed",
			slog.Any("err", err),
		)
	} else {
		slog.Info("cron: scheduled session cleanup", slog.String("schedule", "@daily"))
	}

	cr.Start()
	defer cr.Stop()

	detailers := make([]sources.DetailFetcher, 0, len(srcs))
	for _, s := range srcs {
		if df, ok := s.(sources.DetailFetcher); ok {
			detailers = append(detailers, df)
		}
	}

	slog.Info("queue processing worker starting")
	if err := worker.Run(ctx, q, func(ctx context.Context, qj dto.QueuedJob) error {
		job, err := sources.Dispatch(ctx, detailers, qj.URL)
		if err != nil {
			slog.Error("dispatch failed", slog.String("url", qj.URL), slog.Any("err", err))
			return err
		}
		return exporter.Export(ctx, job)
	}); err != nil {
		slog.Error("worker failed",
			slog.Any("err", err),
		)
	}
}
