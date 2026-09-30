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
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/trace"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

func newJSONLogger(t *testing.T, level string) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	log, err := logger.New(&buf, "json", level)
	if err != nil {
		t.Fatalf("New(json, %q) err = %v", level, err)
	}
	return log, &buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("Unmarshal(%s) err = %v", line, err)
		}
		lines = append(lines, m)
	}
	return lines
}

func newMiddleware(t *testing.T) (http.Handler, *bytes.Buffer) {
	t.Helper()
	log, buf := newJSONLogger(t, "debug")
	h := logger.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "handled")
	}))
	return middleware.RequestID(h), buf
}

func TestNew(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		tests := []struct{ name, format, level string }{
			{name: "empty format and level default to json/info"},
			{name: "text format", format: "text", level: "debug"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var buf bytes.Buffer
				if _, err := logger.New(&buf, tt.format, tt.level); err != nil {
					t.Errorf("New(%q, %q) err = %v", tt.format, tt.level, err)
				}
			})
		}
	})

	t.Run("rejected", func(t *testing.T) {
		tests := []struct{ name, format, level string }{
			{name: "unknown format", format: "xml"},
			{name: "unknown level", level: "trace"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var buf bytes.Buffer
				if _, err := logger.New(&buf, tt.format, tt.level); err == nil {
					t.Errorf("New(%q, %q) err = nil, want error", tt.format, tt.level)
				}
			})
		}
	})
}

func TestNewLevel(t *testing.T) {
	log, buf := newJSONLogger(t, "info")
	log.Debug("below level")
	log.Info("at level")

	lines := logLines(t, buf)
	if len(lines) != 1 || lines[0]["msg"] != "at level" {
		t.Errorf("log lines at LOG_LEVEL=info = %v, want only the INFO line", lines)
	}
}

func TestWithOverride(t *testing.T) {
	log, buf := newJSONLogger(t, "debug")

	ctx := logger.With(t.Context(), slog.String("run_id", "r1"), slog.String(logger.KeySource, "wis"))
	log.InfoContext(ctx, "task", slog.String(logger.KeySource, "override"))

	lines := logLines(t, buf)
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1", len(lines))
	}
	if lines[0]["run_id"] != "r1" {
		t.Errorf("run_id = %v, want r1", lines[0]["run_id"])
	}
	if lines[0][logger.KeySource] != "override" {
		t.Errorf("%s = %v, want override", logger.KeySource, lines[0][logger.KeySource])
	}
	if n := strings.Count(buf.String(), `"`+logger.KeySource+`":`); n != 1 {
		t.Errorf("%q appears %d times, want once", logger.KeySource, n)
	}
}

func TestMiddleware(t *testing.T) {
	t.Run("run ID header and generated IDs", func(t *testing.T) {
		h, buf := newMiddleware(t)
		req := httptest.NewRequest(http.MethodPost, "/ingest", nil)
		req.Header.Set(logger.HeaderRunID, "run-42")
		h.ServeHTTP(httptest.NewRecorder(), req)

		got := logLines(t, buf)[0]
		if got[logger.KeyRunID] != "run-42" {
			t.Errorf("%s = %v, want run-42", logger.KeyRunID, got[logger.KeyRunID])
		}
		for _, key := range []string{logger.KeyRequestID, logger.KeyTraceID} {
			if id, ok := got[key].(string); !ok || id == "" {
				t.Errorf("%s = %v, want a non-empty string", key, got[key])
			}
		}
	})

	t.Run("trace ID differs per request", func(t *testing.T) {
		h, buf := newMiddleware(t)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		lines := logLines(t, buf)
		if len(lines) != 2 {
			t.Fatalf("got %d log lines, want 2", len(lines))
		}
		if lines[0][logger.KeyTraceID] == lines[1][logger.KeyTraceID] {
			t.Errorf("%s repeated across requests: %v", logger.KeyTraceID, lines[0][logger.KeyTraceID])
		}
	})

	t.Run("no run ID header omits the key", func(t *testing.T) {
		h, buf := newMiddleware(t)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		if got := logLines(t, buf)[0]; got[logger.KeyRunID] != nil {
			t.Errorf("%s = %v with no request header, want absent", logger.KeyRunID, got[logger.KeyRunID])
		}
	})
}

func TestTransport(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{name: "run ID in context is sent as a header", ctx: logger.With(t.Context(), slog.String(logger.KeyRunID, "r9")), want: "r9"},
		{name: "no run ID sends no header", ctx: t.Context()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotHeader string
			base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				gotHeader = req.Header.Get(logger.HeaderRunID)
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			})
			req, err := http.NewRequestWithContext(tt.ctx, http.MethodGet, "http://example.com", nil)
			if err != nil {
				t.Fatalf("NewRequestWithContext() err = %v", err)
			}
			if _, err := logger.Transport(base).RoundTrip(req); err != nil {
				t.Fatalf("RoundTrip() err = %v", err)
			}
			if diff := cmp.Diff(tt.want, gotHeader); diff != "" {
				t.Errorf("%s header (-want +got):\n%s", logger.HeaderRunID, diff)
			}
		})
	}
}

func TestSpanContextOverridesTraceID(t *testing.T) {
	log, buf := newJSONLogger(t, "debug")

	traceID := trace.TraceID{1, 2, 3}
	spanID := trace.SpanID{4, 5}
	ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
	ctx = logger.With(ctx, slog.String(logger.KeyTraceID, "uuid-fallback"))
	log.InfoContext(ctx, "handled")

	got := logLines(t, buf)[0]
	if got[logger.KeyTraceID] != traceID.String() {
		t.Errorf("%s = %v, want %s", logger.KeyTraceID, got[logger.KeyTraceID], traceID)
	}
	if got[logger.KeySpanID] != spanID.String() {
		t.Errorf("%s = %v, want %s", logger.KeySpanID, got[logger.KeySpanID], spanID)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
