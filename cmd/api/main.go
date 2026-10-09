package main

import (
	"context"
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

	"github.com/ollymarsters/job-scraper/internal/api"
	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

func main() {
	slog.SetDefault(logger.MustFromEnv())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	for _, key := range []string{"SESSION_SECRET", "INGEST_SERVICE_TOKEN"} {
		if os.Getenv(key) == "" {
			fatal(ctx, "config invalid", fmt.Errorf("%s is required", key))
		}
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

	q, err := queue.NewBrokerFromEnv()
	if err != nil {
		fatal(ctx, "queue init failed", err)
	}
	slog.InfoContext(ctx, "queue client ready")
	defer func() { _ = q.Close() }()

	apps := applications.New(pool)
	idm, err := identity.New(pool, apps,
		os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("GOOGLE_REDIRECT_URL"))
	if err != nil {
		fatal(ctx, "identity init failed", err)
	}

	go schedule.Every(ctx, "session cleanup", 24*time.Hour, idm.DeleteExpiredSessions)

	notifyFrom := os.Getenv("NOTIFY_EMAIL_FROM")
	if notifyFrom == "" {
		notifyFrom = "onboarding@resend.dev"
	}
	js := jobsearch.New(pool, q, scoring.NewFacade(pool), maxAutomaticTargets(ctx))
	scoringModule := scoring.New(pool, idm, idm, js, os.Getenv("RESEND_API_KEY"), notifyFrom,
		scoring.VAPID{PublicKey: os.Getenv("VAPID_PUBLIC_KEY"), PrivateKey: os.Getenv("VAPID_PRIVATE_KEY"), Subject: os.Getenv("VAPID_SUBJECT")})
	go func() {
		if err := scoringModule.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "answer effect loop failed", slog.Any(logger.KeyErr, err))
		}
	}()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(telemetry.NewStateCollector(scoringModule))
	telemetry.ServeMetrics(ctx, reg)

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	cvTemplates := cvtemplates.New(pool, idm.DocsClient())

	cvtailorModule := cvtailor.New(pool, idm.DocsClient(), scoringModule, idm)
	go func() {
		if err := cvtailorModule.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "draft generation loop failed", slog.Any(logger.KeyErr, err))
		}
	}()

	srv := &http.Server{Addr: port, Handler: api.NewRouter(idm, js, apps, cvTemplates, scoringModule, cvtailorModule)}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.InfoContext(ctx, "api server starting", slog.String("addr", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fatal(ctx, "server error", err)
	}
}

func fatal(ctx context.Context, msg string, err error) {
	slog.ErrorContext(ctx, msg, slog.Any(logger.KeyErr, err)) //nolint:sloglint // pedantic: msg is a literal at every call site
	os.Exit(1)
}

func maxAutomaticTargets(ctx context.Context) int {
	raw := os.Getenv("MAX_AUTOMATIC_TARGETS")
	if raw == "" {
		return sourcetargets.DefaultMaxAutomatic
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		fatal(ctx, "config invalid", fmt.Errorf("MAX_AUTOMATIC_TARGETS must be a positive integer, got %q", raw))
	}
	return n
}
