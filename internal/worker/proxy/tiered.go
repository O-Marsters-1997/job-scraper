package proxy

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

const residentialAttempts = 3

type sessionKey struct{}

type tieredTransport struct {
	residential http.RoundTripper
	unlocker    http.RoundTripper
	cache       Cache
}

func newTiered(cache Cache) (*tieredTransport, error) {
	u, err := proxyURL(residentialEnvKey)
	if err != nil {
		return nil, err
	}
	unlocker, err := Transport(true)
	if err != nil {
		return nil, err
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DisableKeepAlives = true
	tr.Proxy = func(req *http.Request) (*url.URL, error) {
		session, _ := req.Context().Value(sessionKey{}).(string)
		pu := *u
		password, _ := u.User.Password()
		pu.User = url.UserPassword(u.User.Username()+"-session-"+session, password)
		return &pu, nil
	}
	return &tieredTransport{
		residential: tr,
		unlocker:    &fetchTransport{base: unlocker, zone: sharedZone},
		cache:       cache,
	}, nil
}

func (t *tieredTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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

func (t *tieredTransport) fetch(req *http.Request) (*http.Response, error) {
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
	return t.unlocker.RoundTrip(req)
}

func (t *tieredTransport) residentialAttempt(req *http.Request) (*http.Response, error) {
	release, err := acquire(req.Context(), req.URL.Hostname())
	if err != nil {
		return nil, err
	}
	session := strconv.FormatUint(rand.Uint64(), 36)
	attempt := req.WithContext(context.WithValue(req.Context(), sessionKey{}, session))
	start := time.Now()
	resp, err := t.residential.RoundTrip(attempt)
	logger.LogFetch(req.Context(), req.URL.String(), resp, err, time.Since(start))
	if err != nil {
		release()
		return nil, err
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
