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

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/robfig/cron/v3"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
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
	noScrape := flag.Bool("no-scrape", false, "skip new scheduled Board checks")
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
	if !*noScrape {
		publishBoards := func() {
			var boards []dto.BoardPoll
			var err error
			if *forceBoards {
				boards, err = js.Boards().ListActiveBoards(ctx)
			} else {
				boards, err = js.Boards().ListDueBoards(ctx)
			}
			if err != nil {
				slog.ErrorContext(ctx, "list Boards failed", slog.Any(logger.KeyErr, err))
				return
			}
			for _, board := range boards {
				task := queue.Task{Version: 1, ID: uuid.NewString(), Source: board.Source, Kind: queue.BoardCheckTask, BoardID: board.ID, Manual: *forceBoards}
				if err := q.Publish(ctx, task); err != nil {
					slog.ErrorContext(ctx, "publish Board check failed", slog.String(logger.KeyBoardID, board.ID), slog.Any(logger.KeyErr, err))
				}
			}
		}
		go publishBoards()
		if _, err := cr.AddFunc(sources.DefaultSchedule, publishBoards); err != nil {
			fatal(ctx, "Board schedule failed", err)
		}
	}
	reconcile := func() {
		targets, err := js.Targets().ListRecoverableSourceTargets(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "list recoverable runs failed", slog.Any(logger.KeyErr, err))
			return
		}
		for _, target := range targets {
			target, err = js.Targets().ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID)
			if errors.Is(err, data.ErrNotFound) {
				continue
			}
			if err != nil {
				slog.ErrorContext(ctx, "claim recoverable run failed", slog.String(logger.KeyTargetID, target.ID), slog.Any(logger.KeyErr, err))
				continue
			}
			task := queue.Task{Version: 1, ID: uuid.NewString(), Source: target.Source, TargetID: target.ID, RunID: target.RunID, Recovery: true}
			if role, _ := sourcespec.SourceRole(target.Source); role == sourcespec.RoleATS {
				boardID, err := js.Boards().GetVerifiedBoardID(ctx, target.Source, target.Value)
				if err != nil {
					slog.ErrorContext(ctx, "recover Board run failed", slog.String(logger.KeyTargetID, target.ID), slog.Any(logger.KeyErr, err))
					continue
				}
				task.Kind, task.BoardID, task.Manual = queue.BoardCheckTask, boardID, true
			} else {
				task.Kind = queue.ListingPageTask
			}
			if err := q.Publish(ctx, task); err != nil {
				slog.ErrorContext(ctx, "recover run publish failed", slog.String(logger.KeyTargetID, target.ID), slog.Any(logger.KeyErr, err))
			}
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
