// Package telemetry ships metrics and logs to Grafana Cloud via Alloy (ADR 0010).
// The Go binaries hold no Grafana credentials and import no vendor SDK.
package telemetry

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const shutdownTimeout = 5 * time.Second

// ServeMetrics runs Serve on METRICS_ADDR (default :9091) until ctx is cancelled.
func ServeMetrics(ctx context.Context, reg *prometheus.Registry) error {
	addr := os.Getenv("METRICS_ADDR")
	if addr == "" {
		addr = ":9091"
	}
	return Serve(ctx, addr, reg)
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
