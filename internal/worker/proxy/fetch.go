package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

const maxBodyBytes = 8 << 20

var errZoneExhausted = errors.New("web unlocker zone exhausted (client_10100)")
var errZonePaused = errors.New("web unlocker zone paused until daily probe")

func IsZonePaused(err error) bool {
	return errors.Is(err, errZoneExhausted) || errors.Is(err, errZonePaused)
}

var sharedZone = &ZoneGate{}
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

type ZoneGate struct {
	mu        sync.Mutex
	paused    bool
	lastProbe time.Time
	Now       func() time.Time
}

func (z *ZoneGate) enter() (bool, error) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if !z.paused {
		return false, nil
	}
	now := time.Now().UTC()
	if z.Now != nil {
		now = z.Now()
	}
	if now.Year() == z.lastProbe.Year() && now.YearDay() == z.lastProbe.YearDay() {
		return false, errZonePaused
	}
	z.lastProbe = now
	return true, nil
}

func (z *ZoneGate) result(exhausted, success, probe bool) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if exhausted {
		z.paused = true
		z.lastProbe = time.Now().UTC()
		if z.Now != nil {
			z.lastProbe = z.Now()
		}
	} else if success && probe {
		z.paused = false
	}
}

type Cache interface {
	LookupFetch(ctx context.Context, url string) (dto.CachedResponse, bool, error)
	PutFetch(ctx context.Context, resp dto.CachedResponse) error
}

var sharedCache Cache

// SetCache must be called before any proxied Fetcher is built.
func SetCache(c Cache) { sharedCache = c }

type fetchTransport struct {
	base  http.RoundTripper
	zone  *ZoneGate
	cache Cache
}

func cachedResponse(req *http.Request, c dto.CachedResponse) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", c.Status, http.StatusText(c.Status)),
		StatusCode:    c.Status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        c.Header,
		Body:          io.NopCloser(bytes.NewReader(c.Body)),
		ContentLength: int64(len(c.Body)),
		Request:       req,
	}
}

func storable(resp *http.Response) bool {
	ok := resp.StatusCode == http.StatusOK || (resp.StatusCode >= 300 && resp.StatusCode < 400)
	return ok && resp.Header.Get("X-Brd-Error") == "" && !strings.Contains(strings.ToLower(resp.Header.Get("Proxy-Status")), "error=")
}

func lookupCached(ctx context.Context, cache Cache, req *http.Request) (*Collector, *http.Response, error) {
	if cache == nil {
		return nil, nil, nil
	}
	collector, _ := ctx.Value(collectorKey{}).(*Collector)
	if collector == nil {
		return nil, nil, nil
	}
	hit, found, err := cache.LookupFetch(ctx, req.URL.String())
	if err != nil {
		return nil, nil, fmt.Errorf("fetch cache lookup: %w", err)
	}
	if !found {
		return collector, nil, nil
	}
	collector.add(req.URL.String())
	return collector, cachedResponse(req, hit), nil
}

func storeResponse(cache Cache, req *http.Request, resp *http.Response, collector *Collector) (*http.Response, error) {
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	stored := dto.CachedResponse{URL: req.URL.String(), Status: resp.StatusCode, Header: resp.Header, Body: body}
	if err := cache.PutFetch(req.Context(), stored); err != nil {
		slog.ErrorContext(req.Context(), "fetch cache write failed", slog.String(logger.KeyURL, stored.URL), slog.Any(logger.KeyErr, err))
	} else {
		collector.add(stored.URL)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func (f *fetchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := ValidateURL(req.Context(), req.URL); err != nil {
		return nil, err
	}
	collector, hit, err := lookupCached(req.Context(), f.cache, req)
	if err != nil || hit != nil {
		return hit, err
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
			resp.Body = &releasingBody{ReadCloser: http.MaxBytesReader(nil, resp.Body, maxBodyBytes), release: release}
			held = false
			if collector != nil && storable(resp) {
				return storeResponse(f.cache, req, resp, collector)
			}
			return resp, nil
		}
		_ = resp.Body.Close()
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(200 * time.Millisecond):
		}
		if req, err = rewound(req); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("unreachable")
}

func rewound(req *http.Request) (*http.Request, error) {
	if req.GetBody == nil {
		return req, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.Body = body
	return req, nil
}

func zoneExhausted(resp *http.Response) bool {
	return resp.Header.Get("X-Brd-Err-Code") == "client_10100" || strings.Contains(resp.Header.Get("Proxy-Status"), "client_10100")
}

type releasingBody struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (b *releasingBody) Close() error {
	err := b.ReadCloser.Close()
	if b.release != nil {
		b.once.Do(b.release)
	}
	return err
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

// Route is how a Source's requests leave the worker. Neither proxy route ever falls back to direct.
type Route int

const (
	Direct Route = iota
	// Unlocker sends every request through Bright Data Web Unlocker.
	Unlocker
	// Tiered tries Decodo residential first, then Unlocker (docs/adr/0018-hostile-sources-fetch-residential-first.md).
	Tiered
)

func Fetcher(route Route) (http.RoundTripper, error) {
	switch route {
	case Unlocker:
		base, err := Transport(true)
		if err != nil {
			return nil, err
		}
		return &fetchTransport{base: base, zone: sharedZone, cache: sharedCache}, nil
	case Tiered:
		return newTiered(sharedCache)
	case Direct:
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
	tr, err := Fetcher(Unlocker)
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

func NewFetchTransport(base http.RoundTripper, zone *ZoneGate, cache Cache) http.RoundTripper {
	return &fetchTransport{base: base, zone: zone, cache: cache}
}
