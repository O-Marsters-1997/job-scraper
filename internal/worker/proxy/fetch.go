package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

const maxBodyBytes = 8 << 20

var errZoneExhausted = errors.New("web unlocker zone exhausted (client_10100)")
var errZonePaused = errors.New("web unlocker zone paused until daily probe")

func IsZonePaused(err error) bool {
	return errors.Is(err, errZoneExhausted) || errors.Is(err, errZonePaused)
}

var sharedZone = &zoneGate{}
var allSlots = make(chan struct{}, 16)
var hostSlots sync.Map

func acquire(ctx context.Context, host string) (func(), error) {
	value, _ := hostSlots.LoadOrStore(host, make(chan struct{}, 2))
	hostSlot := value.(chan struct{})
	select {
	case allSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case hostSlot <- struct{}{}:
		return func() { <-hostSlot; <-allSlots }, nil
	case <-ctx.Done():
		<-allSlots
		return nil, ctx.Err()
	}
}

type zoneGate struct {
	mu        sync.Mutex
	paused    bool
	lastProbe time.Time
	now       func() time.Time
}

func (z *zoneGate) enter() (bool, error) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if !z.paused {
		return false, nil
	}
	now := time.Now().UTC()
	if z.now != nil {
		now = z.now()
	}
	if now.Year() == z.lastProbe.Year() && now.YearDay() == z.lastProbe.YearDay() {
		return false, errZonePaused
	}
	z.lastProbe = now
	return true, nil
}

func (z *zoneGate) result(exhausted, success, probe bool) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if exhausted {
		z.paused = true
		z.lastProbe = time.Now().UTC()
		if z.now != nil {
			z.lastProbe = z.now()
		}
	} else if success && probe {
		z.paused = false
	}
}

type fetchTransport struct {
	base http.RoundTripper
	zone *zoneGate
}

func (f *fetchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := ValidateURL(req.Context(), req.URL); err != nil {
		return nil, err
	}
	probe := false
	if f.zone != nil {
		var err error
		probe, err = f.zone.enter()
		if err != nil {
			return nil, err
		}
	}
	release, err := acquire(req.Context(), req.URL.Hostname())
	if err != nil {
		return nil, err
	}
	held := true
	defer func() {
		if held {
			release()
		}
	}()
	for attempt := 0; attempt < 2; attempt++ {
		start := time.Now()
		resp, err := f.base.RoundTrip(req)
		logger.LogFetch(req.Context(), req.URL.String(), resp, err, time.Since(start))
		if err != nil {
			if f.zone != nil && errors.Is(err, errZoneExhausted) {
				f.zone.result(true, false, probe)
			}
			return nil, err
		}
		if f.zone != nil {
			exhausted := zoneExhausted(resp)
			f.zone.result(exhausted, resp.StatusCode == http.StatusOK, probe)
			if exhausted {
				_ = resp.Body.Close()
				return nil, errZoneExhausted
			}
		}
		if resp.StatusCode != http.StatusTooManyRequests || attempt == 1 {
			resp.Body = &limitedBody{ReadCloser: resp.Body, release: release}
			held = false
			return resp, nil
		}
		_ = resp.Body.Close()
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(200 * time.Millisecond):
		}
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.Body = body
		}
	}
	return nil, errors.New("unreachable")
}

func zoneExhausted(resp *http.Response) bool {
	return resp.Header.Get("X-Brd-Err-Code") == "client_10100" || strings.Contains(resp.Header.Get("Proxy-Status"), "client_10100")
}

type limitedBody struct {
	io.ReadCloser
	n       int64
	release func()
	once    sync.Once
}

func (b *limitedBody) Close() error {
	err := b.ReadCloser.Close()
	if b.release != nil {
		b.once.Do(b.release)
	}
	return err
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.n > maxBodyBytes {
		return 0, errors.New("response exceeds 8 MiB")
	}
	remaining := maxBodyBytes + 1 - b.n
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := b.ReadCloser.Read(p)
	b.n += int64(n)
	if b.n > maxBodyBytes {
		return n, errors.New("response exceeds 8 MiB")
	}
	return n, err
}

func ValidateURL(ctx context.Context, u *url.URL) error {
	_, err := publicAddrs(ctx, u)
	return err
}

func publicAddrs(ctx context.Context, u *url.URL) ([]netip.Addr, error) {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return nil, errors.New("unsafe fetch URL")
	}
	if port := u.Port(); port != "" && port != "80" && port != "443" {
		return nil, errors.New("unsafe fetch port")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil {
		if !publicIP(ip) {
			return nil, fmt.Errorf("unsafe fetch address %s", ip)
		}
		return []netip.Addr{ip}, nil
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("resolve fetch host: %w", err)
	}
	if len(ips) == 0 {
		return nil, errors.New("fetch host has no addresses")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return nil, fmt.Errorf("unsafe fetch address %s", ip)
		}
	}
	return ips, nil
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range []netip.Prefix{
		netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("2001:db8::/32"),
	} {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func Fetcher(useProxy bool) (http.RoundTripper, error) {
	if useProxy {
		base, err := Transport(true)
		if err != nil {
			return nil, err
		}
		return &fetchTransport{base: base, zone: sharedZone}, nil
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := publicAddrs(ctx, &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)})
		if err != nil {
			return nil, err
		}
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	return &fetchTransport{base: tr}, nil
}

// Probe checks whether a paused Web Unlocker zone is usable again.
// Bright Data recommends geo.brdtest.com as its proxy test target
// (https://docs.brightdata.com/proxy-networks/errorCatalog).
func Probe(ctx context.Context) error {
	sharedZone.mu.Lock()
	paused := sharedZone.paused
	sharedZone.mu.Unlock()
	if !paused {
		return nil
	}
	tr, err := Fetcher(true)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: tr}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://geo.brdtest.com/welcome.txt", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("web unlocker probe status %s", resp.Status)
	}
	return nil
}
