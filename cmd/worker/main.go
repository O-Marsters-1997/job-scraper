package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/robfig/cron/v3"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
	"github.com/ollymarsters/job-scraper/internal/worker"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/crawl"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/getro"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/yc"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
)

func main() {
	forceBoards := flag.Bool("scrape-now", false, "check active verified Boards without shifting cadence")
	flag.Parse()
	slog.SetDefault(logger.MustFromEnv())
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := proxy.Validate(); err != nil {
		fatal(ctx, "Web Unlocker config invalid", err)
	}
	shutdownTracing, err := telemetry.InitTracing(ctx)
	if err != nil {
		fatal(ctx, "tracing init failed", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	pool, err := db.Connect(ctx)
	if err != nil {
		fatal(ctx, "db init failed", err)
	}
	defer pool.Close()

	scoringModule := scoring.NewFacade(pool)

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	telemetry.ServeMetrics(ctx, reg)

	q, err := queue.NewBrokerFromEnv()
	if err != nil {
		fatal(ctx, "queue init failed", err)
	}
	defer func() { _ = q.Close() }()

	js := jobsearch.New(pool, q, scoringModule)

	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		fatal(ctx, "config invalid", errors.New("API_BASE_URL is required"))
	}
	exporter := scraper.NewAPIExporter(apiBaseURL, os.Getenv("INGEST_SERVICE_TOKEN"))
	boardPoller := scraper.NewBoardPoller(js.Boards(), scraper.SourceBoardFetcher{}, exporter)
	orch := scraper.New(js.Boards(), scoringModule, builder.BuildSource, js.Targets())
	processor := worker.NewProcessor(worker.Deps{
		JS: js, Broker: q, Orchestrator: orch, Boards: boardPoller, Exporter: exporter,
		Detailers: map[string]sources.DetailFetcher{
			"wis": wis.New(wis.Search{}), "linkedin": linkedin.New("", nil), "indeed": indeed.New(""),
		},
	})
	cr := cron.New()
	publishBoards := func() {
		if err := js.PublishBoardChecks(ctx, *forceBoards); err != nil {
			slog.ErrorContext(ctx, "list Boards failed", slog.Any(logger.KeyErr, err))
		}
	}
	go publishBoards()
	if _, err := cr.AddFunc(sources.DefaultSchedule, publishBoards); err != nil {
		fatal(ctx, "Board schedule failed", err)
	}
	reconcile := func() {
		if err := js.RecoverRuns(ctx); err != nil {
			slog.ErrorContext(ctx, "list recoverable runs failed", slog.Any(logger.KeyErr, err))
		}
	}
	go reconcile()
	if _, err := cr.AddFunc("@every 1m", reconcile); err != nil {
		fatal(ctx, "reconcile schedule failed", err)
	}
	go schedule.Every(ctx, "proxy probe", 24*time.Hour, proxy.Probe)
	go schedule.Every(ctx, "candidate cleanup", 24*time.Hour, js.DeleteExpiredCandidates)
	cr.Start()
	defer cr.Stop()

	go discover.NewRunner([]discover.Harvester{yc.New(), getro.New()}, js.Boards(), js.Boards()).Run(ctx)
	go crawl.New(js.Boards()).Run(ctx)
	slog.InfoContext(ctx, "RabbitMQ source workers starting")
	if err := q.Consume(ctx, processor.Process, processor.FailRun); err != nil && ctx.Err() == nil {
		slog.ErrorContext(ctx, "worker failed", slog.Any(logger.KeyErr, err))
	}
}

func fatal(ctx context.Context, msg string, err error) {
	slog.ErrorContext(ctx, msg, slog.Any(logger.KeyErr, err)) //nolint:sloglint // pedantic: msg is a literal at every call site
	os.Exit(1)
}
