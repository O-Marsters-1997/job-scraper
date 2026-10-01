package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
	"github.com/ollymarsters/job-scraper/internal/worker"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/commoncrawl"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/untracked"
	wttjharvest "github.com/ollymarsters/job-scraper/internal/worker/discover/wttj"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
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
	proxy.SetCache(js)

	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		fatal(ctx, "config invalid", errors.New("API_BASE_URL is required"))
	}
	maxPages := 0
	if raw := os.Getenv("SCRAPE_MAX_PAGES"); raw != "" {
		if maxPages, err = strconv.Atoi(raw); err != nil || maxPages < 0 {
			fatal(ctx, "config invalid", fmt.Errorf("SCRAPE_MAX_PAGES must be a non-negative integer, got %q", raw))
		}
	}
	exporter := scraper.NewAPIExporter(apiBaseURL, os.Getenv("INGEST_SERVICE_TOKEN"))
	boardPoller := scraper.NewBoardPoller(js.Boards(), scraper.SourceBoardFetcher{Profiles: js}, exporter)
	orch := scraper.New(js.Boards(), scoringModule, builder.BuildSource, js.Targets())
	processor := worker.NewProcessor(worker.Deps{
		JS: js, Broker: q, Orchestrator: orch, Boards: boardPoller, Exporter: exporter, MaxPages: maxPages, Scoring: scoringModule, Discover: scraper.DiscoverBoard,
		Detailers: builder.Detailers(), CardComplete: builder.CardComplete(),
	})

	go schedule.Every(ctx, "board checks", time.Hour, func(ctx context.Context) error {
		return js.PublishBoardChecks(ctx, *forceBoards)
	})
	go schedule.Every(ctx, "reconcile", time.Minute, js.RecoverRuns)
	go schedule.Every(ctx, "proxy probe", 24*time.Hour, proxy.Probe)
	go schedule.Every(ctx, "candidate cleanup", 24*time.Hour, js.DeleteExpiredCandidates)
	go schedule.Every(ctx, "fetch cache cleanup", 24*time.Hour, js.DeleteExpiredFetches)

	harvesters := []discover.Harvester{
		commoncrawl.New(&http.Client{Timeout: 2 * time.Minute}, commoncrawl.CollinfoURL),
		wttjharvest.New(&http.Client{Timeout: time.Minute}, wttjharvest.SitemapURL),
		untracked.New(js.Boards()),
	}
	harvest := discover.NewRunner(harvesters, q, js.Boards(), js.Boards())
	go schedule.Every(ctx, "harvest", time.Hour, harvest.RunOnce)
	slog.InfoContext(ctx, "RabbitMQ source workers starting")

	if err := q.Consume(ctx, processor.Process, processor.FailRun); err != nil && ctx.Err() == nil {
		slog.ErrorContext(ctx, "worker failed", slog.Any(logger.KeyErr, err))
	}
}

func fatal(ctx context.Context, msg string, err error) {
	slog.ErrorContext(ctx, msg, slog.Any(logger.KeyErr, err)) //nolint:sloglint // pedantic: msg is a literal at every call site
	os.Exit(1)
}
