package proxy_test

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

func connectEcho(w http.ResponseWriter, _ *http.Request) {
	conn, buf, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		return
	}
	_, _ = io.Copy(conn, buf)
}

func proxyEnv(upstream *fakeProxy) string {
	return "http://res:pw@" + strings.TrimPrefix(upstream.URL, "http://")
}

func newTunnel(t *testing.T) *proxy.Tunnel {
	t.Helper()
	tun, err := proxy.NewTunnel(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tun.Close(); err != nil {
			t.Error(err)
		}
	})
	return tun
}

func roundTripThrough(t *testing.T, tun *proxy.Tunnel) {
	t.Helper()
	conn, err := new(net.Dialer).DialContext(t.Context(), "tcp", tun.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("CONNECT status = %d, want 200", resp.StatusCode)
	}
	if _, err := io.WriteString(conn, "ping"); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(br, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "ping" {
		t.Fatalf("echo = %q, want %q", got, "ping")
	}
}

func TestTunnel(t *testing.T) {
	t.Run("keeps one session until rotated", func(t *testing.T) {
		upstream := newFakeProxy(t, connectEcho)
		t.Setenv("DECODO_PROXY_URL", proxyEnv(upstream))
		tun := newTunnel(t)

		roundTripThrough(t, tun)
		roundTripThrough(t, tun)
		tun.Rotate(t.Context(), "challenge")
		roundTripThrough(t, tun)

		upstream.mu.Lock()
		s := slices.Clone(upstream.sessions)
		upstream.mu.Unlock()
		if len(s) != 3 {
			t.Fatalf("upstream CONNECTs = %d, want 3", len(s))
		}
		if !strings.HasPrefix(s[0], "res-session-") {
			t.Fatalf("username %q lacks a session ID", s[0])
		}
		if s[0] != s[1] {
			t.Errorf("sessions before Rotate = %q, %q, want equal", s[0], s[1])
		}
		if s[2] == s[1] {
			t.Errorf("session after Rotate = %q, want a new one", s[2])
		}
	})

	t.Run("counts wire bytes against the source on the residential route", func(t *testing.T) {
		t.Setenv("DECODO_PROXY_URL", proxyEnv(newFakeProxy(t, connectEcho)))
		roundTripThrough(t, newTunnel(t))
		for _, direction := range []string{"read", "write"} {
			counter := proxy.FetchWireBytes.WithLabelValues(t.Name(), "residential", direction)
			if got := testutil.ToFloat64(counter); got <= 0 {
				t.Errorf("%s wire bytes = %v, want > 0", direction, got)
			}
		}
	})

	t.Run("rejects a missing or invalid proxy URL", func(t *testing.T) {
		for _, raw := range []string{"", "socks5://res:pw@host:1"} {
			t.Setenv("DECODO_PROXY_URL", raw)
			if tun, err := proxy.NewTunnel(t.Name()); err == nil {
				t.Errorf("NewTunnel() with DECODO_PROXY_URL=%q = nil error, want one", raw)
				if err := tun.Close(); err != nil {
					t.Error(err)
				}
			}
		}
	})

	t.Run("close stops the listener", func(t *testing.T) {
		t.Setenv("DECODO_PROXY_URL", proxyEnv(newFakeProxy(t, connectEcho)))
		tun, err := proxy.NewTunnel(t.Name())
		if err != nil {
			t.Fatal(err)
		}
		if err := tun.Close(); err != nil {
			t.Fatalf("Close() = %v", err)
		}
		if conn, err := new(net.Dialer).DialContext(t.Context(), "tcp", tun.Addr()); err == nil {
			_ = conn.Close()
			t.Fatal("dial after Close succeeded, want refused")
		}
	})
}
