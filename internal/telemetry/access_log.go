package telemetry

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// AccessLog emits an EventHTTPRequest log line per request, skipping OPTIONS.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)

		slog.Info("http request",
			slog.String("event", EventHTTPRequest),
			slog.String("method", r.Method),
			slog.String("route", chi.RouteContext(r.Context()).RoutePattern()),
			slog.Int("status", ww.Status()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
	})
}
