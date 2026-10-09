package proxy

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

const residentialAttempts = 3

// ErrBlocked is returned when every residential attempt for a request was blocked.
var ErrBlocked = errors.New("residential fetch blocked on every attempt")

type sessionKey struct{}

type residentialTransport struct {
	residential *http.Transport
	cache       Cache

	session   atomic.Value
	rotations atomic.Int64
	source    string
}

func newSession() string {
	return strconv.FormatUint(rand.Uint64(), 36)
}

func (t *residentialTransport) rotate(ctx context.Context, stale, reason string, status int) {
	if t.session.CompareAndSwap(stale, newSession()) {
		slog.InfoContext(ctx, "residential session rotated",
			slog.String(logger.KeySource, t.source),
			slog.String(logger.KeyReason, reason),
			slog.Int(logger.KeyStatus, status),
			slog.Int64(logger.KeyCount, t.rotations.Add(1)),
		)
	}
	t.residential.CloseIdleConnections()
}

func newResidential(cache Cache, source string) (*residentialTransport, error) {
	u, err := proxyURL(residentialEnvKey)
	if err != nil {
		return nil, err
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = func(req *http.Request) (*url.URL, error) {
		session, _ := req.Context().Value(sessionKey{}).(string)
		pu := *u
		password, _ := u.User.Password()
		pu.User = url.UserPassword(u.User.Username()+"-session-"+session, password)
		return &pu, nil
	}
	t := &residentialTransport{
		residential: tr,
		cache:       cache,
		source:      source,
	}
	t.session.Store(newSession())
	return t, nil
}

func (t *residentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := ValidateURL(req.Context(), req.URL); err != nil {
		return nil, err
	}
	collector, hit, err := lookupCached(req.Context(), t.cache, req)
	if err != nil || hit != nil {
		return hit, err
	}
	resp, err := t.fetch(req)
	if err != nil {
		return nil, err
	}
	if collector != nil && storable(resp) {
		return storeResponse(t.cache, req, resp, collector)
	}
	return resp, nil
}

func (t *residentialTransport) fetch(req *http.Request) (*http.Response, error) {
	for range residentialAttempts {
		resp, err := t.residentialAttempt(req)
		switch {
		case err == nil && !blocked(req, resp):
			return resp, nil
		case err == nil:
			_ = resp.Body.Close()
		case req.Context().Err() != nil:
			return nil, req.Context().Err()
		}
		if req, err = rewound(req); err != nil {
			return nil, err
		}
	}
	return nil, ErrBlocked
}

func (t *residentialTransport) residentialAttempt(req *http.Request) (*http.Response, error) {
	release, err := acquire(req.Context(), req.URL.Hostname())
	if err != nil {
		return nil, err
	}
	session := t.session.Load().(string)
	attempt := req.WithContext(context.WithValue(req.Context(), sessionKey{}, session))
	start := time.Now()
	resp, err := t.residential.RoundTrip(attempt)
	logger.LogFetch(req.Context(), req.URL.String(), resp, err, time.Since(start))
	record(req, t.source, routeResidential, resp, err)
	if err != nil {
		if req.Context().Err() == nil {
			t.rotate(req.Context(), session, "error", 0)
		}
		release()
		return nil, err
	}
	if blocked(req, resp) {
		t.rotate(req.Context(), session, "blocked", resp.StatusCode)
	}
	resp.Body = &releasingBody{ReadCloser: http.MaxBytesReader(nil, resp.Body, maxBodyBytes), release: release}
	return resp, nil
}

func blocked(req *http.Request, resp *http.Response) bool {
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusForbidden, 999:
		return true
	}
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return false
	}
	loc, err := req.URL.Parse(resp.Header.Get("Location"))
	if err != nil {
		return false
	}
	host := strings.ToLower(loc.Hostname())
	if host != "linkedin.com" && !strings.HasSuffix(host, ".linkedin.com") {
		return false
	}
	return strings.HasPrefix(loc.Path, "/authwall") || strings.HasPrefix(loc.Path, "/login")
}
