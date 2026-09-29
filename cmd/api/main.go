package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/ollymarsters/job-scraper/internal/api"
	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/tailoring"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

func main() {
	slog.SetDefault(logger.MustFromEnv())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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

	brokerURL := os.Getenv("RABBITMQ_URL")
	if brokerURL == "" {
		brokerURL = "amqp://guest:guest@localhost:5672/"
	}
	q, err := queue.NewBroker(brokerURL)
	if err != nil {
		slog.ErrorContext(ctx, "queue init failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
	slog.InfoContext(ctx, "queue client ready")
	defer func() { _ = q.Close() }()

	apps := applications.New(pool)
	idm, err := identity.New(pool, apps,
		os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("GOOGLE_REDIRECT_URL"))
	if err != nil {
		slog.ErrorContext(ctx, "identity init failed", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}

	notifyFrom := os.Getenv("NOTIFY_EMAIL_FROM")
	if notifyFrom == "" {
		notifyFrom = "onboarding@resend.dev"
	}
	js := jobsearch.New(pool, q, scoring.NewFacade(pool))
	scoringModule := scoring.New(pool, idm, idm, js, os.Getenv("RESEND_API_KEY"), notifyFrom)
	go func() {
		if err := scoringModule.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "answer effect loop failed", slog.Any(logger.KeyErr, err))
		}
	}()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(telemetry.NewStateCollector(scoringModule))
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9091"
	}
	go func() {
		if err := telemetry.Serve(ctx, metricsAddr, reg); err != nil {
			slog.ErrorContext(ctx, "metrics server failed", slog.Any(logger.KeyErr, err))
		}
	}()

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	cvTemplates := cvtemplates.New(pool, idm.DocsClient())

	tailoringModule := tailoring.New(pool)

	srv := &http.Server{Addr: port, Handler: api.NewRouter(idm, js, apps, cvTemplates, scoringModule, tailoringModule)}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.InfoContext(ctx, "api server starting", slog.String("addr", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.ErrorContext(ctx, "server error", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
}
