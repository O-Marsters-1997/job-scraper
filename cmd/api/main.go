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
	"github.com/ollymarsters/job-scraper/internal/api/credstore"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/candidates"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
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

	cs, err := credstore.New(db)
	if err != nil {
		slog.Error("credstore init failed", slog.Any("err", err))
		os.Exit(1)
	}
	notifyFrom := os.Getenv("NOTIFY_EMAIL_FROM")
	if notifyFrom == "" {
		notifyFrom = "onboarding@resend.dev"
	}
	candidateService := candidates.New(db, q)
	scoringModule := scoring.New(db.Pool(), cs, db, candidateService, os.Getenv("RESEND_API_KEY"), notifyFrom)
	go func() {
		if err := scoringModule.Run(ctx); err != nil {
			slog.Error("answer effect loop failed", slog.Any("err", err))
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
			slog.Error("metrics server failed", slog.Any("err", err))
		}
	}()

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	apps := applications.New(db.Pool())
	idm := identity.New(db.Pool(), apps)

	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		jobsdb.NewGoogleTokenStore(db),
	)
	cvTemplates := cvtemplates.New(db.Pool(), googleClient)

	js := jobsearch.New(db.Pool(), q, scoringModule)

	srv := &http.Server{Addr: port, Handler: api.NewRouter(db, cs, idm, js, apps, cvTemplates, scoringModule)}

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
