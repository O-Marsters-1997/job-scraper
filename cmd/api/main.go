package main

import (
	"context"
	"errors"
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
	"golang.org/x/sync/errgroup"

	"github.com/ollymarsters/job-scraper/internal/api"
	"github.com/ollymarsters/job-scraper/internal/client"
	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/schedule"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
	"github.com/ollymarsters/job-scraper/scripts/migrations"
)

const shutdownTimeout = 20 * time.Second

func main() {
	slog.SetDefault(logger.MustFromEnv())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := run(ctx)
	cancel()
	if err != nil {
		slog.Error("api exited", slog.Any(logger.KeyErr, err))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	for _, key := range []string{"SESSION_SECRET", "INGEST_SERVICE_TOKEN"} {
		if os.Getenv(key) == "" {
			return fmt.Errorf("config invalid: %s is required", key)
		}
	}
	maxTargets, err := maxAutomaticTargets()
	if err != nil {
		return err
	}

	shutdownTracing, err := telemetry.InitTracing(ctx)
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	pool, err := db.Connect(ctx)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer pool.Close()

	if err := db.CheckSchemaNotAhead(ctx, pool, migrations.FS); err != nil {
		return err
	}
	if err := db.CheckSchemaNotBehind(ctx, pool, migrations.FS); err != nil {
		return err
	}

	q, err := queue.NewBrokerFromEnv()
	if err != nil {
		return fmt.Errorf("init queue: %w", err)
	}
	slog.InfoContext(ctx, "queue client ready")
	defer func() { _ = q.Close() }()

	scoringFacade := scoring.NewFacade(pool)
	eventsModule := events.New(pool, scoringFacade)
	apps := applications.New(pool, eventsModule, tasks.NewFacade(pool))
	taskModule := tasks.New(pool, apps)
	idm, err := identity.New(pool, []identity.Seeder{apps, taskModule, cvtailor.NewFacade(pool)}, os.Getenv("SESSION_SECRET"),
		os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("GOOGLE_REDIRECT_URL"))
	if err != nil {
		return fmt.Errorf("init identity: %w", err)
	}

	notifyFrom := os.Getenv("NOTIFY_EMAIL_FROM")
	if notifyFrom == "" {
		notifyFrom = "onboarding@resend.dev"
	}
	var proxyQuota jobsearch.QuotaFetcher
	if key := os.Getenv("DECODO_API_KEY"); key != "" {
		proxyQuota = jobsearch.NewDecodoQuota(&http.Client{Timeout: 10 * time.Second}, jobsearch.DecodoSubscriptionsURL, key)
	}
	js := jobsearch.New(pool, q, scoringFacade, proxyQuota, maxTargets, slog.Default())
	notifyModule := notify.New(pool, notify.Ntfy{URL: os.Getenv("NTFY_URL"), Token: os.Getenv("NTFY_TOKEN")})
	scoringModule := scoring.New(pool, idm, idm, js, js, notifyModule, os.Getenv("RESEND_API_KEY"), notifyFrom,
		scoring.VAPID{PublicKey: os.Getenv("VAPID_PUBLIC_KEY"), PrivateKey: os.Getenv("VAPID_PRIVATE_KEY"), Subject: os.Getenv("VAPID_SUBJECT")})

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(telemetry.NewStateCollector(scoringModule))

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	cvTemplates := cvtemplates.New(pool, idm.DocsClient())
	cvtailorModule := cvtailor.New(pool, idm.DocsClient(), scoringModule, scoringModule, idm, slog.Default())

	router, err := api.NewRouter(idm, js, apps, taskModule, cvTemplates, scoringModule, cvtailorModule, eventsModule, notifyModule)
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}
	srv := &http.Server{
		Addr:              port,
		Handler:           client.Middleware(router),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return schedule.Every(gctx, "session cleanup", 24*time.Hour, idm.DeleteExpiredSessions)
	})
	if proxyQuota != nil {
		g.Go(func() error {
			return schedule.Every(gctx, "proxy usage refresh", jobsearch.ProxyUsageInterval, js.RefreshProxyUsage)
		})
	}
	g.Go(func() error {
		if err := scoringModule.Run(gctx); err != nil {
			return fmt.Errorf("run answer effect loop: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		if err := cvtailorModule.Run(gctx); err != nil {
			return fmt.Errorf("run draft generation loop: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		if err := telemetry.ServeMetrics(gctx, reg); err != nil {
			return fmt.Errorf("serve metrics: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		slog.InfoContext(gctx, "api server starting", slog.String("addr", port))
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http: %w", err)
		}
		return nil
	})
	return g.Wait()
}

func maxAutomaticTargets() (int, error) {
	raw := os.Getenv("MAX_AUTOMATIC_TARGETS")
	if raw == "" {
		return sourcetargets.DefaultMaxAutomatic, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("config invalid: MAX_AUTOMATIC_TARGETS must be a positive integer, got %q", raw)
	}
	return n, nil
}
