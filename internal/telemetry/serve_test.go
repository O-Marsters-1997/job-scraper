package telemetry_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

func TestServeStopsOnContextCancel(t *testing.T) {
	addr := freeAddr(t)
	reg := prometheus.NewRegistry()
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() { done <- telemetry.Serve(ctx, addr, reg) }()

	waitForServing(t, addr)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not stop after context cancel")
	}
}

func TestServeExposesMetrics(t *testing.T) {
	addr := freeAddr(t)
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_metric_total"})
	counter.Inc()
	reg.MustRegister(counter)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = telemetry.Serve(ctx, addr, reg) }()

	body := waitForServing(t, addr)
	if !strings.Contains(body, "test_metric_total 1") {
		t.Fatalf("expected registered metric in response body, got: %s", body)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func waitForServing(t *testing.T, addr string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/metrics")
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			buf := make([]byte, 4096)
			n, _ := resp.Body.Read(buf)
			return string(buf[:n])
		}
		<-ticker.C
	}
	t.Fatal("server never became reachable")
	return ""
}
