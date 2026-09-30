package telemetry_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func accessLogLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
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
		if m[logger.KeyEvent] == telemetry.EventHTTPRequest {
			lines = append(lines, m)
		}
	}
	return lines
}

func serveOnce(t *testing.T, method, pattern, target string, status int) (*httptest.ResponseRecorder, []map[string]any) {
	t.Helper()
	buf := captureLogs(t)
	r := chi.NewRouter()
	r.Use(telemetry.AccessLog)
	r.MethodFunc(method, pattern, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec, accessLogLines(t, buf)
}

func TestAccessLog(t *testing.T) {
	t.Run("emits an http.request event", func(t *testing.T) {
		_, lines := serveOnce(t, http.MethodGet, "/jobs/{id}", "/jobs/123", http.StatusTeapot)
		if len(lines) != 1 {
			t.Fatalf("got %d http.request lines, want 1", len(lines))
		}

		want := map[string]any{
			logger.KeyEvent:  telemetry.EventHTTPRequest,
			logger.KeyMethod: http.MethodGet,
			logger.KeyRoute:  "/jobs/{id}",
			logger.KeyStatus: float64(http.StatusTeapot),
		}
		ignore := cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
			_, wanted := want[k]
			return !wanted
		})
		if diff := cmp.Diff(want, lines[0], ignore); diff != "" {
			t.Errorf("log line (-want +got):\n%s", diff)
		}
		if _, ok := lines[0][logger.KeyDurationMS]; !ok {
			t.Errorf("%s missing from log line: %v", logger.KeyDurationMS, lines[0])
		}
	})

	t.Run("skips OPTIONS but still handles it", func(t *testing.T) {
		rec, lines := serveOnce(t, http.MethodOptions, "/jobs/{id}", "/jobs/123", http.StatusNoContent)
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if len(lines) != 0 {
			t.Errorf("got %d http.request lines for OPTIONS, want 0", len(lines))
		}
	})
}
