package telemetry

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ollymarsters/job-scraper/internal/logger"
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

		slog.InfoContext(r.Context(), "http request",
			slog.String(logger.KeyEvent, EventHTTPRequest),
			slog.String(logger.KeyMethod, r.Method),
			slog.String(logger.KeyRoute, chi.RouteContext(r.Context()).RoutePattern()),
			slog.Int(logger.KeyStatus, ww.Status()),
			slog.Int64(logger.KeyDurationMS, time.Since(start).Milliseconds()),
		)
	})
}
