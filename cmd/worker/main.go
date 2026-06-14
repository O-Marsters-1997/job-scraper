package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/notify"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/ashby"
	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/sources/lever"
	"github.com/ollymarsters/job-scraper/internal/sources/personio"
	"github.com/ollymarsters/job-scraper/internal/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/sources/workable"
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

	notifSvc := setupNotifications()

	srcs := []sources.Source{wis.New()}

	if boards := os.Getenv("GREENHOUSE_BOARDS"); boards != "" {
		tokens := splitBoards(boards)
		srcs = append(srcs, greenhouse.New(greenhouse.Config{Boards: tokens}))
		slog.Info("greenhouse source registered", slog.Int("boards", len(tokens)))
	}

	if boards := os.Getenv("LEVER_BOARDS"); boards != "" {
		tokens := splitBoards(boards)
		srcs = append(srcs, lever.New(lever.Config{Boards: tokens}))
		slog.Info("lever source registered", slog.Int("boards", len(tokens)))
	}

	if boards := os.Getenv("ASHBY_BOARDS"); boards != "" {
		tokens := splitBoards(boards)
		srcs = append(srcs, ashby.New(ashby.Config{Boards: tokens}))
		slog.Info("ashby source registered", slog.Int("boards", len(tokens)))
	}

	if boards := os.Getenv("WORKABLE_BOARDS"); boards != "" {
		tokens := splitBoards(boards)
		srcs = append(srcs, workable.New(workable.Config{Boards: tokens}))
		slog.Info("workable source registered", slog.Int("boards", len(tokens)))
	}

	if boards := os.Getenv("RECRUITEE_BOARDS"); boards != "" {
		tokens := splitBoards(boards)
		srcs = append(srcs, recruitee.New(recruitee.Config{Boards: tokens}))
		slog.Info("recruitee source registered", slog.Int("boards", len(tokens)))
	}

	if boards := os.Getenv("PERSONIO_BOARDS"); boards != "" {
		tokens := splitBoards(boards)
		srcs = append(srcs, personio.New(personio.Config{Boards: tokens}))
		slog.Info("personio source registered", slog.Int("boards", len(tokens)))
	}

	orch := scraper.New(srcs, db, q)

	var ingestScorer *score.IngestScorer
	if scoringUserID := os.Getenv("SCORING_USER_ID"); scoringUserID != "" {
		orch.WithRelevanceGate(score.NewHeuristicScorer(), db, db, scoringUserID)
		slog.Info("relevance gate enabled", slog.String("user_id", scoringUserID))

		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			claudeScorer := score.NewClaudeScorer(score.ClaudeScorerConfig{APIKey: apiKey})
			ingestScorer = score.NewIngestScorer(claudeScorer, db, db, scoringUserID)
			slog.Info("suitability scorer enabled", slog.String("user_id", scoringUserID))
		}
	}

	// Build the ingest seam shared by both the ATS and HTML/queue paths.
	// Use typed interface vars to avoid the nil-concrete-pointer pitfall.
	var scorer ingest.Scorer
	if ingestScorer != nil {
		scorer = ingestScorer
	}
	var notifier ingest.Notifier
	if notifSvc != nil {
		notifier = notifSvc
	}
	ing := ingest.New(db, scorer, notifier)
	orch.WithIngester(ing)
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

	if notifSvc != nil {
		digestSchedule := os.Getenv("NOTIFY_DIGEST_CRON")
		if digestSchedule == "" {
			digestSchedule = "0 9 * * *"
		}
		if _, err := cr.AddFunc(digestSchedule, func() {
			lastSent, err := db.GetLastDigestSentAt(ctx)
			if err != nil {
				slog.Error("digest: get last sent failed",
					slog.Any("err", err),
				)
				return
			}
			var jobs []dto.Job
			if lastSent.IsZero() {
				jobs, err = db.List(ctx)
			} else {
				jobs, err = db.ListSince(ctx, lastSent)
			}
			if err != nil {
				slog.Error("digest: list jobs failed",
					slog.Any("err", err),
				)
				return
			}
			if len(jobs) == 0 {
				slog.Info("digest: no new jobs, skipping")
				return
			}
			if err := notifSvc.SendDigest(ctx, jobs); err != nil {
				slog.Error("digest: send failed",
					slog.Any("err", err),
				)
				return
			}
			if err := db.RecordDigest(ctx, time.Now(), len(jobs)); err != nil {
				slog.Error("digest: record failed",
					slog.Any("err", err),
				)
			}
		}); err != nil {
			slog.Error("digest cron schedule failed",
				slog.Any("err", err),
			)
		} else {
			slog.Info("cron: scheduled digest", slog.String("schedule", digestSchedule))
		}
	}

	cr.Start()
	defer cr.Stop()

	slog.Info("queue processing worker starting")
	if err := worker.Run(ctx, q, func(ctx context.Context, qj dto.QueuedJob) error {
		job, err := sources.Dispatch(ctx, srcs, qj.URL)
		if err != nil {
			slog.Error("dispatch failed", slog.String("url", qj.URL), slog.Any("err", err))
			return err
		}
		return ing.Ingest(ctx, []dto.Job{job})
	}); err != nil {
		slog.Error("worker failed",
			slog.Any("err", err),
		)
	}
}

func splitBoards(env string) []string {
	parts := strings.Split(env, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func setupNotifications() *notify.NotificationService {
	apiKey := os.Getenv("RESEND_API_KEY")
	to := os.Getenv("NOTIFY_EMAIL_TO")
	from := os.Getenv("NOTIFY_EMAIL_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	if apiKey == "" || to == "" {
		slog.Info("notifications disabled: RESEND_API_KEY or NOTIFY_EMAIL_TO not set")
		return nil
	}

	renderer, err := notify.NewRenderer()
	if err != nil {
		slog.Error("notify: failed to load templates",
			slog.Any("err", err),
		)
		return nil
	}

	cfg := notify.Config{
		To:              to,
		OnIngestEnabled: os.Getenv("NOTIFY_ON_INGEST") == "true",
		DigestEnabled:   os.Getenv("NOTIFY_DIGEST_ENABLED") != "false",
	}

	return notify.NewNotificationService(
		notify.NewResendNotifier(apiKey, from),
		renderer,
		cfg,
	)
}
