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

func accessLogLines(buf *bytes.Buffer) []map[string]any {
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		if m["event"] == telemetry.EventHTTPRequest {
			lines = append(lines, m)
		}
	}
	return lines
}

func TestAccessLogEmitsHTTPRequestEvent(t *testing.T) {
	buf := captureLogs(t)

	r := chi.NewRouter()
	r.Use(telemetry.AccessLog)
	r.Get("/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/jobs/123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	lines := accessLogLines(buf)
	if len(lines) != 1 {
		t.Fatalf("got %d http.request lines, want 1 (log: %s)", len(lines), buf.String())
	}

	line := lines[0]
	if line["method"] != http.MethodGet {
		t.Errorf("method = %v, want GET", line["method"])
	}
	if line["route"] != "/jobs/{id}" {
		t.Errorf("route = %v, want /jobs/{id}", line["route"])
	}
	status, _ := line["status"].(float64)
	if int(status) != http.StatusTeapot {
		t.Errorf("status = %v, want %d", line["status"], http.StatusTeapot)
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Errorf("duration_ms missing from log line: %v", line)
	}
}

func TestAccessLogSkipsOptions(t *testing.T) {
	buf := captureLogs(t)

	r := chi.NewRouter()
	r.Use(telemetry.AccessLog)
	r.Options("/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodOptions, "/jobs/123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (OPTIONS handling must still run)", rec.Code, http.StatusNoContent)
	}
	if lines := accessLogLines(buf); len(lines) != 0 {
		t.Errorf("got %d http.request lines for OPTIONS, want 0", len(lines))
	}
}
