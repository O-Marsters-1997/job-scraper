package logger

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

// FetchTransport wraps base with a DEBUG "outbound fetch" line per round
// trip (url, status, duration_ms, err). Safe for a third-party client: it
// never logs headers or bodies.
func FetchTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return OutboundSpans(fetchLogTransport{base})
}

// OutboundSpans records a client span per round trip without sending trace
// headers, so a third party never sees our trace IDs.
func OutboundSpans(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base, otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator()))
}

type fetchLogTransport struct {
	base http.RoundTripper
}

func (t fetchLogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	LogFetch(req.Context(), req.URL.String(), resp, err, time.Since(start))
	return resp, err
}

// LogFetch is FetchTransport's line, for a caller with its own RoundTrip
// (retries, proxy zones) that can't just wrap a base transport.
func LogFetch(ctx context.Context, url string, resp *http.Response, err error, duration time.Duration) {
	if !slog.Default().Enabled(ctx, slog.LevelDebug) {
		return
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	attrs := []any{
		slog.String(KeyURL, url),
		slog.Int(KeyStatus, status),
		slog.Int64(KeyDurationMS, duration.Milliseconds()),
	}
	if err != nil {
		attrs = append(attrs, slog.Any(KeyErr, err))
	}
	slog.DebugContext(ctx, "outbound fetch", attrs...)
}
