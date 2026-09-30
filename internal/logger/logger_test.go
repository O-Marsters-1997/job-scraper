package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		level   string
		wantErr bool
	}{
		{name: "empty format and level default to json/info"},
		{name: "text format", format: "text", level: "debug"},
		{name: "unknown format errors", format: "xml", wantErr: true},
		{name: "unknown level errors", level: "trace", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := logger.New(&buf, tt.format, tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%q, %q) error = %v, wantErr %v", tt.format, tt.level, err, tt.wantErr)
			}
		})
	}
}

func TestNewLevel(t *testing.T) {
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", "info")
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	log.Debug("below level")
	log.Info("at level")

	out := buf.String()
	if strings.Contains(out, "below level") {
		t.Errorf("output contains a DEBUG line at LOG_LEVEL=info: %s", out)
	}
	if !strings.Contains(out, "at level") {
		t.Errorf("output missing the INFO line: %s", out)
	}
}

func TestWithOverride(t *testing.T) {
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", "debug")
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	ctx := logger.With(context.Background(), slog.String("run_id", "r1"), slog.String(logger.KeySource, "wis"))
	log.InfoContext(ctx, "task", slog.String(logger.KeySource, "override"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", buf.String(), err)
	}
	if got["run_id"] != "r1" {
		t.Errorf("run_id = %v, want r1", got["run_id"])
	}
	if got[logger.KeySource] != "override" {
		t.Errorf("%s = %v, want override (call-site should win)", logger.KeySource, got[logger.KeySource])
	}
	if n := strings.Count(buf.String(), `"`+logger.KeySource+`":`); n != 1 {
		t.Errorf("%q appears %d times in %s, want once (no duplicate JSON key)", logger.KeySource, n, buf.String())
	}
}

func TestMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", "debug")
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	handler := logger.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "handled")
	}))
	handler = middleware.RequestID(handler)

	req := httptest.NewRequest(http.MethodPost, "/ingest", nil)
	req.Header.Set(logger.HeaderRunID, "run-42")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", buf.String(), err)
	}
	if got[logger.KeyRunID] != "run-42" {
		t.Errorf("%s = %v, want run-42", logger.KeyRunID, got[logger.KeyRunID])
	}
	if id, ok := got[logger.KeyRequestID].(string); !ok || id == "" {
		t.Errorf("%s = %v, want a non-empty request ID", logger.KeyRequestID, got[logger.KeyRequestID])
	}
	if id, ok := got[logger.KeyTraceID].(string); !ok || id == "" {
		t.Errorf("%s = %v, want a non-empty trace ID", logger.KeyTraceID, got[logger.KeyTraceID])
	}
}

func TestMiddlewareTraceIDPerRequest(t *testing.T) {
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", "debug")
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	handler := logger.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "handled")
	}))
	handler = middleware.RequestID(handler)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	var first, second map[string]any
	lines := strings.SplitN(strings.TrimSpace(buf.String()), "\n", 2)
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2: %s", len(lines), buf.String())
	}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", lines[0], err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", lines[1], err)
	}
	if first[logger.KeyTraceID] == second[logger.KeyTraceID] {
		t.Errorf("%s repeated across requests: %v", logger.KeyTraceID, first[logger.KeyTraceID])
	}
}

func TestMiddlewareNoRunIDHeader(t *testing.T) {
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", "debug")
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	handler := logger.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "handled")
	}))
	handler = middleware.RequestID(handler)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", buf.String(), err)
	}
	if _, ok := got[logger.KeyRunID]; ok {
		t.Errorf("%s present with no request header: %v", logger.KeyRunID, got[logger.KeyRunID])
	}
}

func TestTransport(t *testing.T) {
	var gotHeader string
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		gotHeader = req.Header.Get(logger.HeaderRunID)
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	ctx := logger.With(context.Background(), slog.String(logger.KeyRunID, "r9"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() = %v", err)
	}
	if _, err := logger.Transport(base).RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() = %v", err)
	}
	if gotHeader != "r9" {
		t.Errorf("%s header = %q, want r9", logger.HeaderRunID, gotHeader)
	}
}

func TestTransportNoRunID(t *testing.T) {
	var gotHeader string
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		gotHeader = req.Header.Get(logger.HeaderRunID)
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() = %v", err)
	}
	if _, err := logger.Transport(base).RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() = %v", err)
	}
	if gotHeader != "" {
		t.Errorf("%s header = %q, want none", logger.HeaderRunID, gotHeader)
	}
}

func TestSpanContextOverridesTraceID(t *testing.T) {
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", "debug")
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	traceID := trace.TraceID{1, 2, 3}
	spanID := trace.SpanID{4, 5}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
	ctx = logger.With(ctx, slog.String(logger.KeyTraceID, "uuid-fallback"))
	log.InfoContext(ctx, "handled")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal(%s) = %v", buf.String(), err)
	}
	if got[logger.KeyTraceID] != traceID.String() {
		t.Errorf("%s = %v, want %s", logger.KeyTraceID, got[logger.KeyTraceID], traceID)
	}
	if got[logger.KeySpanID] != spanID.String() {
		t.Errorf("%s = %v, want %s", logger.KeySpanID, got[logger.KeySpanID], spanID)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
