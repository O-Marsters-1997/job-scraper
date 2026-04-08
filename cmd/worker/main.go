package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/notify"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/worker"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := jobsdb.New(ctx, jobsdb.ConnString())
	if err != nil {
		slog.Error("db init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer db.Close()

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}

	q, err := queue.New(valkeyAddr)
	if err != nil {
		slog.Error("queue init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer q.Close()

	notifSvc := setupNotifications()

	srcs := []sources.Source{wis.New()}

	orch := scraper.New(srcs, db, q)
	if err := orch.Start(ctx); err != nil {
		slog.Error("orchestrator start failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer orch.Stop()

	cr := cron.New()

	if _, err := cr.AddFunc("@daily", func() {
		if err := db.DeleteExpiredSessions(ctx); err != nil {
			slog.Error("session cleanup failed", slog.Any("err", err))
		}
	}); err != nil {
		slog.Error("session cleanup cron schedule failed", slog.Any("err", err))
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
				slog.Error("digest: get last sent failed", slog.Any("err", err))
				return
			}
			var jobs []dto.Job
			if lastSent.IsZero() {
				jobs, err = db.List(ctx)
			} else {
				jobs, err = db.ListSince(ctx, lastSent)
			}
			if err != nil {
				slog.Error("digest: list jobs failed", slog.Any("err", err))
				return
			}
			if len(jobs) == 0 {
				slog.Info("digest: no new jobs, skipping")
				return
			}
			if err := notifSvc.SendDigest(ctx, jobs); err != nil {
				slog.Error("digest: send failed", slog.Any("err", err))
				return
			}
			if err := db.RecordDigest(ctx, time.Now(), len(jobs)); err != nil {
				slog.Error("digest: record failed", slog.Any("err", err))
			}
		}); err != nil {
			slog.Error("digest cron schedule failed", slog.Any("err", err))
		} else {
			slog.Info("cron: scheduled digest", slog.String("schedule", digestSchedule))
		}
	}

	cr.Start()
	defer cr.Stop()

	slog.Info("queue processing worker starting")
	if err := worker.Run(ctx, q, func(ctx context.Context, url string) error {
		job, err := sources.Dispatch(ctx, srcs, url)
		if err != nil {
			slog.Error("dispatch failed", slog.String("url", url), slog.Any("err", err))
			return err
		}
		if err := db.Save(ctx, []dto.Job{job}); err != nil {
			slog.Error("upsert failed", slog.String("url", url), slog.Any("err", err))
			return err
		}
		slog.Info("job upserted", slog.String("url", url), slog.String("title", job.Title))
		if notifSvc != nil {
			notifSvc.NotifyNewJob(ctx, job)
		}
		return nil
	}); err != nil {
		slog.Error("worker failed", slog.Any("err", err))
	}
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
		slog.Error("notify: failed to load templates", slog.Any("err", err))
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
