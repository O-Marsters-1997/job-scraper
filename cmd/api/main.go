package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/ollymarsters/job-scraper/internal/api"
	"github.com/ollymarsters/job-scraper/internal/api/credstore"
	"github.com/ollymarsters/job-scraper/internal/api/jev"
	"github.com/ollymarsters/job-scraper/internal/api/notify"
	"github.com/ollymarsters/job-scraper/internal/api/services/suitability"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

// noAlerts is the Alerter used when no RESEND_API_KEY is configured (dev).
type noAlerts struct{}

func (noAlerts) NotifyNewJob(context.Context, dto.Job, string) error { return nil }

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

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(telemetry.NewStateCollector(db))
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9091"
	}
	go func() {
		if err := telemetry.Serve(ctx, metricsAddr, reg); err != nil {
			slog.Error("metrics server failed", slog.Any("err", err))
		}
	}()

	cs, err := credstore.New(db)
	if err != nil {
		slog.Error("credstore init failed", slog.Any("err", err))
		os.Exit(1)
	}
	var alerter suitability.Alerter = noAlerts{}
	if apiKey := os.Getenv("RESEND_API_KEY"); apiKey != "" {
		renderer, err := notify.NewRenderer()
		if err != nil {
			slog.Error("notify templates unavailable", slog.Any("err", err))
		} else {
			from := os.Getenv("NOTIFY_EMAIL_FROM")
			if from == "" {
				from = "onboarding@resend.dev"
			}
			alerter = notify.NewNotificationService(notify.NewResendNotifier(apiKey, from), renderer)
		}
	}
	suitabilitySvc := suitability.New(db, db, db, jev.NewClient(), cs, alerter, db)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := suitabilitySvc.RunTick(ctx); err != nil && ctx.Err() == nil {
					slog.Error("answer effect tick failed", slog.Any("err", err))
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

	srv := &http.Server{Addr: port, Handler: api.NewRouter(db, q, cs, suitabilitySvc)}

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
