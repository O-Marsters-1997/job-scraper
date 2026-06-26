package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const envKey = "BRIGHTDATA_PROXY_URL"

// Transport returns an http.RoundTripper for the requested mode.
// If useProxy is false, returns http.DefaultTransport (direct, no proxy).
// If useProxy is true, reads BRIGHTDATA_PROXY_URL and configures a transport
// routing through BrightData Web Unlocker. If the env var is unset the proxy
// falls back to http.DefaultTransport — degrades gracefully rather than
// crashing on missing creds.
func Transport(useProxy bool) (http.RoundTripper, error) {
	if !useProxy {
		return http.DefaultTransport, nil
	}
	raw := os.Getenv(envKey)
	if raw == "" {
		return http.DefaultTransport, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("proxy: parse %s: %w", envKey, err)
	}
	// ponytail: InsecureSkipVerify is required because Web Unlocker performs
	// TLS interception (MITM) to handle CAPTCHA/JS/fingerprinting. The hop to
	// brd.superproxy.io is authenticated by the zone password and we only
	// fetch public job listings. Upgrade path: add BrightData's CA cert to a
	// custom x509.CertPool and set TLSClientConfig.RootCAs instead.
	return &http.Transport{
		Proxy:           http.ProxyURL(u),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}, nil
}
