package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/discover"
	"github.com/ollymarsters/job-scraper/internal/discover/crawl"
	"github.com/ollymarsters/job-scraper/internal/discover/getro"
	"github.com/ollymarsters/job-scraper/internal/discover/yc"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/worker"
)

func main() {
	forceScrape := flag.Bool("scrape-now", false,
		"check every tracked verified Board on startup and each scheduled tick without shifting cadence")
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

	boardPoller := scraper.NewBoardPoller(db, scraper.SourceBoardFetcher{}, exporter)
	buildOne := func(target dto.SourceTarget) []sources.Source {
		return builder.BuildSources([]dto.SourceTarget{target}, nil)
	}

	orch := scraper.New(nil, db, q).
		WithExporter(exporter).
		WithSourceReloader(nil, buildOne)

	if *forceScrape {
		slog.Info("force scrape enabled: checking all tracked verified Boards")
	}

	orch.WithRejectFilter(db)
	orch.WithCandidates(db)
	slog.Info("reject filter enabled (multi-user)")

	cr := cron.New()
	if !*noScrape {
		poll := boardPoller.PollDue
		if *forceScrape {
			poll = boardPoller.PollAll
		}
		go func() {
			if err := poll(ctx); err != nil {
				slog.Error("board poll failed", slog.Any("err", err))
			}
		}()
		if _, err := cr.AddFunc(sources.DefaultSchedule, func() {
			if err := poll(ctx); err != nil {
				slog.Error("board poll failed", slog.Any("err", err))
			}
		}); err != nil {
			slog.Error("board poll schedule failed", slog.Any("err", err))
			os.Exit(1)
		}
	} else {
		slog.Info("no-scrape enabled: skipping scheduled boards, draining queue only")
	}

	if _, err := cr.AddFunc("@daily", func() {
		if err := db.DeleteExpiredSessions(ctx); err != nil {
			slog.Error("session cleanup failed",
				slog.Any("err", err),
			)
		}
		if err := db.DeleteExpiredCandidates(ctx); err != nil {
			slog.Error("candidate cleanup failed", slog.Any("err", err))
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

	detailers := make([]sources.DetailFetcher, 0, 3)
	// Always include a bare wis scraper for GetDetails so that on-demand scrape
	// job URLs can be fetched even when no wis targets exist at boot.
	detailers = append(detailers, wis.New(wis.Config{}))
	detailers = append(detailers, linkedin.New(linkedin.Config{}), indeed.New(indeed.Config{}))

	go worker.RunScrapeRequests(ctx, q, func(ctx context.Context, req dto.ScrapeRequest) error {
		role, _ := sources.SourceRole(req.Target.Source)
		if role == sources.RoleATS {
			id, err := db.GetVerifiedBoardID(ctx, req.Target.Source, req.Target.Value)
			if err != nil {
				return err
			}
			return boardPoller.PollBoard(ctx, id, true)
		}
		if _, err := db.SetSourceTargetRunState(ctx, req.Target.ID, "running", ""); err != nil {
			return err
		}
		if err := orch.ScrapeTarget(ctx, req.Target); err != nil {
			if _, stateErr := db.SetSourceTargetRunState(ctx, req.Target.ID, "failed", "Search failed. Try running it again."); stateErr != nil {
				slog.Error("record search failure failed", slog.Any("err", stateErr))
			}
			return err
		}
		_, err := db.SetSourceTargetRunState(ctx, req.Target.ID, "succeeded", "")
		return err
	})

	// Company harvester runner: drains code-registered harvesters into the
	// companies catalog on a 24h-per-harvester gate. Catalog-only — no source
	// targets, nothing scrapable results from this.
	harvestRunner := discover.NewRunner([]discover.Harvester{yc.New(), getro.New()}, db, q)
	go harvestRunner.Run(ctx)

	// Careers-page crawler: the self-expanding step. Resolves companies.domain
	// rows with no known ats_source into a board by fetching their careers
	// page directly (never proxied). Still catalog-only — a resolved board
	// isn't scraped until the Companies-page tracking toggle turns it into a
	// source_targets row.
	careersCrawler := crawl.New(db)
	go careersCrawler.Run(ctx)

	// Dead-letter monitor: surfaces standing accumulation so orphaned URLs don't
	// pile up silently. Only logs when the set is non-empty.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := q.DeadLetterCount(ctx); err != nil {
					slog.Error("dead-letter count failed", slog.Any("err", err))
				} else if n > 0 {
					slog.Warn("dead-letter queue non-empty", slog.Int64("count", n))
				}
			}
		}
	}()

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
