package main

import (
	"context"
	"flag"
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
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/worker"
)

func main() {
	forceScrape := flag.Bool("scrape-now", false,
		"bypass per-source recency gate so every tick (startup + cron) scrapes immediately")
	noScrape := flag.Bool("no-scrape", false,
		"skip source enqueueing entirely; only drain URLs already in the queue")
	flag.Parse()

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

	targets, err := db.ListEnabledSourceTargets(ctx)
	if err != nil {
		slog.Error("load source targets failed", slog.Any("err", err))
		os.Exit(1)
	}
	srcs := builder.BuildSources(targets)
	slog.Info("sources built from db", slog.Int("count", len(srcs)))

	buildAll := func(ctx context.Context) ([]sources.Source, error) {
		ts, err := db.ListEnabledSourceTargets(ctx)
		if err != nil {
			return nil, err
		}
		return builder.BuildSources(ts), nil
	}
	buildOne := func(target dto.SourceTarget) []sources.Source {
		return builder.BuildSources([]dto.SourceTarget{target})
	}

	orch := scraper.New(srcs, db, q).
		WithExporter(exporter).
		WithSourceReloader(buildAll, buildOne).
		WithForceScrape(*forceScrape)

	if *forceScrape {
		slog.Info("force scrape enabled: ignoring per-source recency gate")
	}

	orch.WithRelevanceGate(&score.HeuristicScorer{}, db)
	slog.Info("relevance gate enabled (multi-user)")

	if *noScrape {
		slog.Info("no-scrape enabled: skipping source enqueueing, draining queue only")
	} else if err := orch.Start(ctx); err != nil {
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

	detailers := make([]sources.DetailFetcher, 0, len(srcs)+1)
	for _, s := range srcs {
		if df, ok := s.(sources.DetailFetcher); ok {
			detailers = append(detailers, df)
		}
	}
	// Always include a bare wis scraper for GetDetails so that on-demand scrape
	// job URLs can be fetched even when no wis targets exist at boot.
	detailers = append(detailers, wis.New(wis.Config{}))

	// Scrape-request consumer: handles scrape_now requests from the API.
	go worker.RunScrapeRequests(ctx, q, func(ctx context.Context, req dto.ScrapeRequest) error {
		return orch.ScrapeTarget(ctx, req.Target)
	})

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
