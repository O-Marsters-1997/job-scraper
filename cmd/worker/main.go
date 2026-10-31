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
	"golang.org/x/sync/errgroup"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
	"github.com/ollymarsters/job-scraper/internal/worker"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/commoncrawl"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/sluglist"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/untracked"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/vcportfolio"
	wellfoundharvest "github.com/ollymarsters/job-scraper/internal/worker/discover/wellfound"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/workablesearch"
	wttjharvest "github.com/ollymarsters/job-scraper/internal/worker/discover/wttj"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/yc"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wellfound"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

var forceBoards = flag.Bool("scrape-now", false, "check active verified Boards without shifting cadence")

func main() {
	flag.Parse()
	slog.SetDefault(logger.MustFromEnv())
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := run(ctx)
	cancel()
	if err != nil {
		slog.Error("worker exited", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if err := proxy.ValidateResidential(); err != nil {
		return fmt.Errorf("residential proxy config invalid: %w", err)
	}
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		return errors.New("config invalid: API_BASE_URL is required")
	}
	ingestToken := os.Getenv("INGEST_SERVICE_TOKEN")
	if ingestToken == "" {
		return errors.New("config invalid: INGEST_SERVICE_TOKEN is required")
	}
	decodoScrapeToken := os.Getenv("DECODO_SCRAPE_TOKEN")
	if decodoScrapeToken == "" {
		return errors.New("config invalid: DECODO_SCRAPE_TOKEN is required")
	}
	maxPages := 0
	if raw := os.Getenv("SCRAPE_MAX_PAGES"); raw != "" {
		var err error
		if maxPages, err = strconv.Atoi(raw); err != nil || maxPages < 0 {
			return fmt.Errorf("config invalid: SCRAPE_MAX_PAGES must be a non-negative integer, got %q", raw)
		}
	}
	shutdownTracing, err := telemetry.InitTracing(ctx)
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	pool, err := db.Connect(ctx)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer pool.Close()

	scoringModule := scoring.NewFacade(pool)

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	proxy.RegisterMetrics(reg)
	jobsearch.RegisterMetrics(reg)

	q, err := queue.NewBrokerFromEnv()
	if err != nil {
		return fmt.Errorf("init queue: %w", err)
	}
	defer func() { _ = q.Close() }()

	js := jobsearch.New(pool, q, scoringModule, nil, sourcetargets.DefaultMaxAutomatic, slog.Default())
	proxy.SetCache(js)

	exporter := scraper.NewAPIExporter(apiBaseURL, ingestToken).WithLogger(slog.Default())
	wellfoundGate := wellfound.NewGate()
	sourceBuilder := builder.New(wellfoundGate, decodoScrapeToken)
	boardPoller := scraper.NewBoardPoller(js.Boards(), scraper.SourceBoardFetcher{Build: sourceBuilder.BuildSource, Profiles: js}, exporter)
	orch := scraper.New(js.Boards(), scoringModule, sourceBuilder.BuildSource, js.Targets())
	processor := worker.NewProcessor(worker.Deps{
		JS: js, Broker: q, Orchestrator: orch, Boards: boardPoller, Exporter: exporter, MaxPages: maxPages, Scoring: scoringModule, Logger: slog.Default(), Discover: func(ctx context.Context, source, token string) (scraper.Discovery, error) {
			return scraper.DiscoverBoard(ctx, sourceBuilder.BuildSource, source, token)
		},
		Verify: func(ctx context.Context, source, token string) (scraper.Verification, error) {
			return scraper.VerifyBoard(ctx, sourceBuilder.BuildSource, source, token)
		},
		Detailers: sourceBuilder.Detailers(), CardComplete: sourceBuilder.CardComplete(),
	})

	domainBase := sources.NewBase(sources.Config{Name: "domain-resolver"})
	harvesters := []discover.Harvester{
		commoncrawl.New(&http.Client{Timeout: 2 * time.Minute}, commoncrawl.CollinfoURL),
		wttjharvest.New(&http.Client{Timeout: time.Minute}, wttjharvest.SitemapURL, wttj.NewClient(time.Second), js.Boards(), exporter, time.Now),
		untracked.New(js.Boards(), untracked.UK),
		untracked.New(js.Boards(), untracked.Abroad),
		workablesearch.New(&http.Client{Timeout: time.Minute}, workablesearch.SearchURL),
		yc.New(&http.Client{Timeout: time.Minute}, yc.CompaniesURL, domainBase.Get),
		vcportfolio.New(&http.Client{Timeout: time.Minute}, vcportfolio.PortfolioURLs(), domainBase.Get, slog.Default()),
		wellfoundharvest.New(wellfound.NewPages(wellfoundGate), domainBase.Get, js.Boards(), wellfoundharvest.LocationPages),
		sluglist.New(&http.Client{Timeout: time.Minute}, sluglist.Lists),
	}
	harvest := discover.NewRunner(harvesters, q, js.Boards(), js.Boards())

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := telemetry.ServeMetrics(gctx, reg); err != nil {
			return fmt.Errorf("serve metrics: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		return schedule.Every(gctx, "board checks", time.Hour, func(ctx context.Context) error {
			return js.PublishBoardChecks(ctx, *forceBoards)
		})
	})
	g.Go(func() error { return schedule.Every(gctx, "reconcile", time.Minute, js.RecoverRuns) })
	g.Go(func() error { return schedule.Every(gctx, "discovery runs", time.Minute, js.PublishDueTargets) })
	g.Go(func() error {
		return schedule.Every(gctx, "candidate cleanup", 24*time.Hour, js.DeleteExpiredCandidates)
	})
	g.Go(func() error {
		return schedule.Every(gctx, "scrape run cleanup", 24*time.Hour, js.DeleteExpiredScrapeRuns)
	})
	g.Go(func() error {
		return schedule.Every(gctx, "fetch cache cleanup", 24*time.Hour, js.DeleteExpiredFetches)
	})
	g.Go(func() error { return schedule.Every(gctx, "harvest", 15*time.Minute, harvest.RunOnce) })
	g.Go(func() error {
		slog.InfoContext(gctx, "RabbitMQ source workers starting")
		if err := q.Consume(gctx, processor.Process, processor.FailRun); err != nil {
			return fmt.Errorf("consume queue: %w", err)
		}
		return nil
	})
	return g.Wait()
}
