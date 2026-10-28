package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

const (
	tunnelDialTimeout    = 30 * time.Second
	tunnelRequestTimeout = 30 * time.Second
	tunnelAcceptBackoff  = 10 * time.Millisecond
)

type Tunnel struct {
	listener  net.Listener
	upstream  *url.URL
	session   atomic.Pointer[string]
	rotations atomic.Int64
	source    string
	allow     []string
	closing   context.Context
	stop      context.CancelFunc
	handlers  sync.WaitGroup
}

func NewTunnel(source string) (*Tunnel, error) {
	return NewRestrictedTunnel(source)
}

func NewRestrictedTunnel(source string, allowedHosts ...string) (*Tunnel, error) {
	u, err := proxyURL(residentialEnvKey)
	if err != nil {
		return nil, err
	}
	l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen tunnel: %w", err)
	}
	closing, stop := context.WithCancel(context.Background())
	t := &Tunnel{listener: l, upstream: u, source: source, allow: allowedHosts, closing: closing, stop: stop}
	t.renewSession()
	t.handlers.Go(t.serve)
	return t, nil
}

func (t *Tunnel) Addr() string {
	return t.listener.Addr().String()
}

func (t *Tunnel) Rotate(ctx context.Context, reason string) {
	t.renewSession()
	slog.InfoContext(ctx, "residential session rotated", //nolint:forbidigo // NewTunnel's frozen signature (#1020) takes no logger
		slog.String(logger.KeySource, t.source),
		slog.String(logger.KeyReason, reason),
		slog.Int64(logger.KeyCount, t.rotations.Add(1)),
	)
}

func (t *Tunnel) Close() error {
	t.stop()
	err := t.listener.Close()
	t.handlers.Wait()
	return err
}

func (t *Tunnel) renewSession() {
	s := newSession()
	t.session.Store(&s)
}

func (t *Tunnel) serve() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			time.Sleep(tunnelAcceptBackoff)
			continue
		}
		t.handlers.Go(func() { t.handle(conn) })
	}
}

func (t *Tunnel) handle(client net.Conn) {
	defer func() { _ = client.Close() }()
	defer context.AfterFunc(t.closing, func() { _ = client.Close() })()
	br := bufio.NewReader(client)
	if err := client.SetReadDeadline(time.Now().Add(tunnelRequestTimeout)); err != nil {
		return
	}
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	if err := client.SetReadDeadline(time.Time{}); err != nil {
		return
	}
	if req.Method != http.MethodConnect {
		writeStatus(client, http.StatusMethodNotAllowed)
		return
	}
	if !t.permits(req.Host) || !t.permits(req.URL.Host) {
		writeStatus(client, http.StatusForbidden)
		return
	}
	upstream, err := t.dial(req)
	if err != nil {
		writeStatus(client, http.StatusBadGateway)
		return
	}
	defer func() { _ = upstream.Close() }()
	t.handlers.Go(func() {
		_, _ = io.Copy(upstream, br)
		_ = upstream.Close()
	})
	_, _ = io.Copy(client, upstream)
}

func (t *Tunnel) permits(authority string) bool {
	if len(t.allow) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(authority)
	if err != nil {
		host = authority
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return slices.ContainsFunc(t.allow, func(allowed string) bool {
		return host == allowed || strings.HasSuffix(host, "."+allowed)
	})
}

func (t *Tunnel) dial(req *http.Request) (net.Conn, error) {
	d := net.Dialer{Timeout: tunnelDialTimeout}
	conn, err := d.DialContext(t.closing, "tcp", t.upstream.Host)
	if err != nil {
		return nil, fmt.Errorf("dial upstream proxy: %w", err)
	}
	password, _ := t.upstream.User.Password()
	creds := t.upstream.User.Username() + "-session-" + *t.session.Load() + ":" + password
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(creds)))
	counted := newCountingConn(conn, t.source, routeResidential)
	if err := req.Write(counted); err != nil {
		return nil, errors.Join(fmt.Errorf("write connect: %w", err), conn.Close())
	}
	return counted, nil
}

func writeStatus(w io.Writer, code int) {
	_, _ = fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n\r\n", code, http.StatusText(code))
}
