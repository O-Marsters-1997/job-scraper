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
	"sync"
	"sync/atomic"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

const tunnelDialTimeout = 30 * time.Second

type Tunnel struct {
	listener  net.Listener
	upstream  *url.URL
	session   atomic.Pointer[string]
	rotations atomic.Int64
	source    string
	serving   sync.WaitGroup
}

func NewTunnel(source string) (*Tunnel, error) {
	u, err := proxyURL(residentialEnvKey)
	if err != nil {
		return nil, err
	}
	l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen tunnel: %w", err)
	}
	t := &Tunnel{listener: l, upstream: u, source: source}
	t.newSession()
	t.serving.Go(t.serve)
	return t, nil
}

func (t *Tunnel) Addr() string {
	return t.listener.Addr().String()
}

func (t *Tunnel) Rotate(ctx context.Context, reason string) {
	t.newSession()
	slog.InfoContext(ctx, "residential session rotated", //nolint:forbidigo // NewTunnel's frozen signature (#1020) takes no logger
		slog.String(logger.KeySource, t.source),
		slog.String(logger.KeyReason, reason),
		slog.Int64(logger.KeyCount, t.rotations.Add(1)),
	)
}

func (t *Tunnel) Close() error {
	err := t.listener.Close()
	t.serving.Wait()
	return err
}

func (t *Tunnel) newSession() {
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
			continue
		}
		go t.handle(conn)
	}
}

func (t *Tunnel) handle(client net.Conn) {
	defer func() { _ = client.Close() }()
	br := bufio.NewReader(client)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	if req.Method != http.MethodConnect {
		writeStatus(client, http.StatusMethodNotAllowed)
		return
	}
	upstream, err := t.dial(req)
	if err != nil {
		writeStatus(client, http.StatusBadGateway)
		return
	}
	defer func() { _ = upstream.Close() }()
	go func() {
		_, _ = io.Copy(upstream, br)
		_ = upstream.Close()
	}()
	_, _ = io.Copy(client, upstream)
}

func (t *Tunnel) dial(req *http.Request) (net.Conn, error) {
	d := net.Dialer{Timeout: tunnelDialTimeout}
	conn, err := d.DialContext(req.Context(), "tcp", t.upstream.Host)
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
