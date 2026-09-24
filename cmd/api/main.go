package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"

	app "github.com/ollymarsters/job-scraper/internal"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/notify"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
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
		slog.Error("db init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer db.Close()

	cs, err := credstore.New(db)
	if err != nil {
		slog.Error("credstore init failed", slog.Any("err", err))
		os.Exit(1)
	}
	var sendAlert func(context.Context, dto.Job, string) error
	if apiKey := os.Getenv("RESEND_API_KEY"); apiKey != "" {
		renderer, err := notify.NewRenderer()
		if err != nil {
			slog.Error("notify templates unavailable", slog.Any("err", err))
		} else {
			from := os.Getenv("NOTIFY_EMAIL_FROM")
			if from == "" {
				from = "onboarding@resend.dev"
			}
			sendAlert = notify.NewNotificationService(notify.NewResendNotifier(apiKey, from), renderer).NotifyNewJob
		}
	}
	outbox := score.NewOutboxWorker(db,
		func(ctx context.Context, userID string) (string, error) { return cs.Get(ctx, userID, "anthropic") },
		func(apiKey string) score.SuitabilityScorer {
			return score.NewClaudeScorer(score.ClaudeScorerConfig{APIKey: apiKey})
		},
		sendAlert,
	)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := outbox.RunOnce(ctx); err != nil && !errors.Is(err, pgx.ErrNoRows) && ctx.Err() == nil {
					slog.Error("scoring effect failed", slog.Any("err", err))
				}
			}
		}
	}()

	brokerURL := os.Getenv("RABBITMQ_URL")
	if brokerURL == "" {
		brokerURL = "amqp://guest:guest@localhost:5672/"
	}
	q, err := queue.NewBroker(brokerURL)
	if err != nil {
		slog.Error("queue init failed", slog.Any("err", err))
		os.Exit(1)
	}
	slog.Info("queue client ready")
	defer func() { _ = q.Close() }()

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	srv := &http.Server{Addr: port, Handler: app.NewRouter(ctx, db, q, cs)}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.Info("api server starting", slog.String("addr", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("err", err))
		os.Exit(1)
	}
}
