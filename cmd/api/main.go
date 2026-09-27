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
	"github.com/ollymarsters/job-scraper/internal/api/jev"
	"github.com/ollymarsters/job-scraper/internal/api/notify"
	"github.com/ollymarsters/job-scraper/internal/api/services/suitability"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
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
		if err := suitabilitySvc.Run(ctx); err != nil {
			slog.Error("answer effect loop failed", slog.Any("err", err))
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

	apps := applications.New(db.Pool())
	idm := identity.New(db.Pool(), apps)

	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		jobsdb.NewGoogleTokenStore(db),
	)
	cvTemplates := cvtemplates.New(db.Pool(), googleClient)

	js := jobsearch.New(db.Pool(), q, db)

	srv := &http.Server{Addr: port, Handler: api.NewRouter(db, q, cs, suitabilitySvc, idm, js, apps, cvTemplates)}

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
