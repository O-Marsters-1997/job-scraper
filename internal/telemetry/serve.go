// Package telemetry ships metrics and logs to Grafana Cloud via Alloy (ADR 0010).
// The Go binaries hold no Grafana credentials and import no vendor SDK.
package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

// ServeMetrics runs Serve in the background on METRICS_ADDR (default :9091), logging a failure.
func ServeMetrics(ctx context.Context, reg *prometheus.Registry) {
	addr := os.Getenv("METRICS_ADDR")
	if addr == "" {
		addr = ":9091"
	}
	go func() {
		if err := Serve(ctx, addr, reg); err != nil {
			slog.ErrorContext(ctx, "metrics server failed", slog.Any(logger.KeyErr, err))
		}
	}()
}

// Serve runs a dedicated HTTP server exposing reg at /metrics on addr, stopping when ctx is cancelled.
func Serve(ctx context.Context, addr string, reg *prometheus.Registry) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	srv := &http.Server{Addr: addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
