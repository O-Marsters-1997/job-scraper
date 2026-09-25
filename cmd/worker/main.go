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
	"github.com/robfig/cron/v3"

	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/discover"
	"github.com/ollymarsters/job-scraper/internal/discover/crawl"
	"github.com/ollymarsters/job-scraper/internal/discover/getro"
	"github.com/ollymarsters/job-scraper/internal/discover/yc"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/proxy"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

func main() {
	forceBoards := flag.Bool("scrape-now", false, "check active verified Boards without shifting cadence")
	noScrape := flag.Bool("no-scrape", false, "skip new scheduled Board checks")
	flag.Parse()
	slog.SetDefault(logger.New())
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := proxy.Validate(); err != nil {
		slog.Error("Web Unlocker config invalid", slog.Any("err", err))
		os.Exit(1)
	}
	connStr, err := jobsdb.ConnString()
	if err != nil {
		slog.Error("db config invalid", slog.Any("err", err))
		os.Exit(1)
	}
	db, err := jobsdb.New(ctx, connStr)
	if err != nil {
		slog.Error("db init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer db.Close()
	brokerURL := os.Getenv("RABBITMQ_URL")
	if brokerURL == "" {
		brokerURL = "amqp://guest:guest@localhost:5672/"
	}
	q, err := queue.NewBroker(brokerURL)
	if err != nil {
		slog.Error("queue init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = q.Close() }()
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		slog.Error("API_BASE_URL is required")
		os.Exit(1)
	}
	exporter := scraper.NewAPIExporter(apiBaseURL, os.Getenv("INGEST_SERVICE_TOKEN"))
	boardPoller := scraper.NewBoardPoller(db, scraper.SourceBoardFetcher{}, exporter)
	orch := scraper.New(db, q).WithSourceBuilder(
		func(target dto.SourceTarget) []sources.Source {
			return builder.BuildSources([]dto.SourceTarget{target})
		})
	orch.WithRejectFilter(db)
	orch.WithCandidates(db)
	processor := &taskProcessor{
		db: db, broker: q, orchestrator: orch, boards: boardPoller, exporter: exporter,
		detailers: map[string]sources.DetailFetcher{
			"wis": wis.New(wis.Config{}), "linkedin": linkedin.New(linkedin.Config{}), "indeed": indeed.New(indeed.Config{}),
		},
	}
	cr := cron.New()
	if !*noScrape {
		publishBoards := func() {
			var boards []dto.BoardPoll
			var err error
			if *forceBoards {
				boards, err = db.ListActiveBoards(ctx)
			} else {
				boards, err = db.ListDueBoards(ctx)
			}
			if err != nil {
				slog.Error("list Boards failed", slog.Any("err", err))
				return
			}
			for _, board := range boards {
				task := queue.Task{Version: 1, ID: uuid.NewString(), Source: board.Source, Kind: queue.BoardCheckTask, BoardID: board.ID, Manual: *forceBoards}
				if err := q.Publish(ctx, task); err != nil {
					slog.Error("publish Board check failed", slog.String("board_id", board.ID), slog.Any("err", err))
				}
			}
		}
		go publishBoards()
		if _, err := cr.AddFunc(sources.DefaultSchedule, publishBoards); err != nil {
			slog.Error("Board schedule failed", slog.Any("err", err))
			os.Exit(1)
		}
	}
	reconcile := func() {
		targets, err := db.ListRecoverableSourceTargets(ctx)
		if err != nil {
			slog.Error("list recoverable runs failed", slog.Any("err", err))
			return
		}
		for _, target := range targets {
			target, err = db.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID)
			if errors.Is(err, providers.ErrNotFound) {
				continue
			}
			if err != nil {
				slog.Error("claim recoverable run failed", slog.String("target_id", target.ID), slog.Any("err", err))
				continue
			}
			task := queue.Task{Version: 1, ID: uuid.NewString(), Source: target.Source, TargetID: target.ID, RunID: target.RunID, Recovery: true}
			if role, _ := sourcespec.SourceRole(target.Source); role == sourcespec.RoleATS {
				boardID, err := db.GetVerifiedBoardID(ctx, target.Source, target.Value)
				if err != nil {
					slog.Error("recover Board run failed", slog.String("target_id", target.ID), slog.Any("err", err))
					continue
				}
				task.Kind, task.BoardID, task.Manual = queue.BoardCheckTask, boardID, true
			} else {
				task.Kind = queue.ListingPageTask
			}
			if err := q.Publish(ctx, task); err != nil {
				slog.Error("recover run publish failed", slog.String("target_id", target.ID), slog.Any("err", err))
			}
		}
	}
	go reconcile()
	if _, err := cr.AddFunc("@every 1m", reconcile); err != nil {
		slog.Error("reconcile schedule failed", slog.Any("err", err))
		os.Exit(1)
	}
	if _, err := cr.AddFunc("@daily", func() {
		if err := proxy.Probe(ctx); err != nil {
			slog.Warn("Web Unlocker daily probe failed", slog.Any("err", err))
		}
		if err := db.DeleteExpiredSessions(ctx); err != nil {
			slog.Error("session cleanup failed", slog.Any("err", err))
		}
		if err := db.DeleteExpiredCandidates(ctx); err != nil {
			slog.Error("candidate cleanup failed", slog.Any("err", err))
		}
	}); err != nil {
		slog.Error("daily schedule failed", slog.Any("err", err))
		os.Exit(1)
	}
	cr.Start()
	defer cr.Stop()

	go discover.NewRunner([]discover.Harvester{yc.New(), getro.New()}, db, db).Run(ctx)
	go crawl.New(db).Run(ctx)
	slog.Info("RabbitMQ source workers starting")
	if err := q.Consume(ctx, processor.process, processor.failRun); err != nil && ctx.Err() == nil {
		slog.Error("worker failed", slog.Any("err", err))
	}
}

func (p *taskProcessor) failRun(ctx context.Context, task queue.Task) error {
	if task.Kind == queue.DetailTask || task.TargetID == "" || task.RunID == "" {
		return nil
	}
	_, err := p.db.TransitionSourceTargetRun(ctx, task.TargetID, task.RunID, "failed", "Work failed after retries. Try running it again.")
	if errors.Is(err, providers.ErrNotFound) {
		return nil
	}
	return err
}

type taskProcessor struct {
	db           *jobsdb.DB
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
		slog.Warn("board verification failed", slog.String("company_id", task.CompanyID), slog.String("source", task.Source), slog.String("token", task.BoardToken), slog.Any("err", err))
		return nil
	}
	_, err := p.db.VerifyCompanyBoard(ctx, task.CompanyID, task.Source, task.BoardToken, "user_confirmed")
	return err
}

func (p *taskProcessor) currentTarget(ctx context.Context, task queue.Task) (dto.SourceTarget, bool, error) {
	target, err := p.db.GetSourceTarget(ctx, task.TargetID)
	if errors.Is(err, providers.ErrNotFound) {
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
		if _, err := p.db.TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", ""); err != nil {
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
		_, err = p.db.TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", "")
		return err
	}
	_, err = p.db.TransitionSourceTargetRun(ctx, target.ID, task.RunID, "succeeded", "")
	return err
}

func (p *taskProcessor) processBoard(ctx context.Context, task queue.Task) error {
	if task.TargetID != "" {
		target, active, err := p.currentTarget(ctx, task)
		if err != nil || !active {
			return err
		}
		if _, err := p.db.TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", ""); err != nil {
			return err
		}
	}
	if err := p.boards.PollBoard(ctx, task.BoardID, task.Manual); err != nil {
		return err
	}
	if task.TargetID != "" {
		_, err := p.db.TransitionSourceTargetRun(ctx, task.TargetID, task.RunID, "succeeded", "")
		return err
	}
	return nil
}
