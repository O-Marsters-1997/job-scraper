package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
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
		slog.ErrorContext(ctx, "Web Unlocker config invalid", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	shutdownTracing, err := telemetry.InitTracing(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "tracing init failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	pool, err := db.Connect(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "db init failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	defer pool.Close()

	apps := applications.New(pool)
	idm := identity.NewFacade(pool, apps)
	scoringModule := scoring.NewFacade(pool)

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9091"
	}
	go func() {
		if err := telemetry.Serve(ctx, metricsAddr, reg); err != nil {
			slog.ErrorContext(ctx, "metrics server failed", slog.Any(logger.KeyErr, err))
		}
	}()

	brokerURL := os.Getenv("RABBITMQ_URL")
	if brokerURL == "" {
		brokerURL = "amqp://guest:guest@localhost:5672/"
	}
	q, err := queue.NewBroker(brokerURL)
	if err != nil {
		slog.ErrorContext(ctx, "queue init failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	defer func() { _ = q.Close() }()

	js := jobsearch.New(pool, q, scoringModule)

	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		slog.ErrorContext(ctx, "API_BASE_URL is required")
		os.Exit(1)
	}
	exporter := scraper.NewAPIExporter(apiBaseURL, os.Getenv("INGEST_SERVICE_TOKEN"))
	boardPoller := scraper.NewBoardPoller(js.Boards(), scraper.SourceBoardFetcher{}, exporter)
	orch := scraper.New(js.Catalog()).WithSourceBuilder(builder.BuildSource)
	orch.WithRejectFilter(scoringModule)
	orch.WithCandidates(js.Targets())
	processor := &taskProcessor{
		js: js, broker: q, orchestrator: orch, boards: boardPoller, exporter: exporter,
		detailers: map[string]sources.DetailFetcher{
			"wis": wis.New(wis.Search{}), "linkedin": linkedin.New(linkedin.Search{}), "indeed": indeed.New(""),
		},
	}
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
			slog.ErrorContext(ctx, "Board schedule failed", slog.Any(logger.KeyErr, err))
			os.Exit(1)
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
		slog.ErrorContext(ctx, "reconcile schedule failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	if _, err := cr.AddFunc("@daily", func() {
		if err := proxy.Probe(ctx); err != nil {
			slog.WarnContext(ctx, "Web Unlocker daily probe failed", slog.Any(logger.KeyErr, err))
		}
		if err := idm.DeleteExpiredSessions(ctx); err != nil {
			slog.ErrorContext(ctx, "session cleanup failed", slog.Any(logger.KeyErr, err))
		}
		if err := js.DeleteExpiredCandidates(ctx); err != nil {
			slog.ErrorContext(ctx, "candidate cleanup failed", slog.Any(logger.KeyErr, err))
		}
	}); err != nil {
		slog.ErrorContext(ctx, "daily schedule failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	cr.Start()
	defer cr.Stop()

	go discover.NewRunner([]discover.Harvester{yc.New(), getro.New()}, js.Boards(), js.Boards()).Run(ctx)
	go crawl.New(js.Boards()).Run(ctx)
	slog.InfoContext(ctx, "RabbitMQ source workers starting")
	if err := q.Consume(ctx, processor.process, processor.failRun); err != nil && ctx.Err() == nil {
		slog.ErrorContext(ctx, "worker failed", slog.Any(logger.KeyErr, err))
	}
}

func (p *taskProcessor) failRun(ctx context.Context, task queue.Task) error {
	if task.Kind == queue.DetailTask || task.TargetID == "" || task.RunID == "" {
		return nil
	}
	_, err := p.js.Targets().TransitionSourceTargetRun(ctx, task.TargetID, task.RunID, "failed", "Work failed after retries. Try running it again.")
	if errors.Is(err, data.ErrNotFound) {
		return nil
	}
	return err
}

type taskProcessor struct {
	js           *jobsearch.Module
	broker       *queue.Broker
	orchestrator *scraper.Orchestrator
	boards       *scraper.BoardPoller
	detailers    map[string]sources.DetailFetcher
	exporter     *scraper.APIExporter
}

func (p *taskProcessor) process(ctx context.Context, task queue.Task) error {
	switch task.Kind {
	case queue.DetailTask:
		if task.Source == "remoteok" || task.Source == "remotive" {
			return p.exporter.Export(ctx, task.Card)
		}
		fetcher := p.detailers[task.Source]
		if fetcher == nil {
			return fmt.Errorf("source %s cannot fetch details", task.Source)
		}
		job, err := fetcher.GetDetails(ctx, task.URL)
		if err != nil {
			return err
		}
		return p.exporter.Export(ctx, job)
	case queue.ListingPageTask:
		return p.processPage(ctx, task)
	case queue.BoardCheckTask:
		return p.processBoard(ctx, task)
	case queue.BoardVerifyTask:
		return p.verifyBoard(ctx, task)
	default:
		return fmt.Errorf("unsupported task kind %s", task.Kind)
	}
}

func (p *taskProcessor) verifyBoard(ctx context.Context, task queue.Task) error {
	if err := scraper.VerifyBoard(ctx, task.Source, task.BoardToken); err != nil {
		slog.WarnContext(ctx, "board verification failed", slog.String(logger.KeyCompanyID, task.CompanyID), slog.String(logger.KeySource, task.Source), slog.String("token", task.BoardToken), slog.Any(logger.KeyErr, err))
		return nil
	}
	_, err := p.js.Boards().VerifyCompanyBoard(ctx, task.CompanyID, task.Source, task.BoardToken, "user_confirmed")
	return err
}

func (p *taskProcessor) currentTarget(ctx context.Context, task queue.Task) (dto.SourceTarget, bool, error) {
	target, err := p.js.Targets().GetSourceTarget(ctx, task.TargetID)
	if errors.Is(err, data.ErrNotFound) {
		return dto.SourceTarget{}, false, nil
	}
	if err != nil {
		return dto.SourceTarget{}, false, err
	}
	if !target.Enabled || target.RunID != task.RunID || target.Source != task.Source || target.RunStatus == "succeeded" || target.RunStatus == "failed" {
		return target, false, nil
	}
	if task.Cursor == "" && !task.Redelivered && !task.Recovery && target.RunStatus == "running" && time.Since(target.UpdatedAt) < 30*time.Minute {
		return target, false, nil
	}
	return target, true, nil
}

func (p *taskProcessor) processPage(ctx context.Context, task queue.Task) error {
	target, active, err := p.currentTarget(ctx, task)
	if err != nil || !active {
		return err
	}
	if target.RunStatus == "queued" || task.Cursor == "" {
		if _, err := p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", ""); err != nil {
			return err
		}
	}
	next, err := p.orchestrator.ScrapePage(ctx, target, task.Cursor)
	if err != nil {
		return err
	}
	if next != "" {
		task.ID, task.Cursor = uuid.NewString(), next
		if err := p.broker.Publish(ctx, task); err != nil {
			return err
		}
		_, err = p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", "")
		return err
	}
	_, err = p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "succeeded", "")
	return err
}

func (p *taskProcessor) processBoard(ctx context.Context, task queue.Task) error {
	if task.TargetID != "" {
		target, active, err := p.currentTarget(ctx, task)
		if err != nil || !active {
			return err
		}
		if _, err := p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", ""); err != nil {
			return err
		}
	}
	if err := p.boards.PollBoard(ctx, task.BoardID, task.Manual); err != nil {
		return err
	}
	if task.TargetID != "" {
		_, err := p.js.Targets().TransitionSourceTargetRun(ctx, task.TargetID, task.RunID, "succeeded", "")
		return err
	}
	return nil
}
