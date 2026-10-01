package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

const (
	envKey            = "BRIGHTDATA_PROXY_URL"
	residentialEnvKey = "DECODO_PROXY_URL"
)

func Validate() error {
	_, err := Transport(true)
	return err
}

// ValidateResidential checks DECODO_PROXY_URL, which every tiered Source needs.
func ValidateResidential() error {
	_, err := proxyURL(residentialEnvKey)
	return err
}

func proxyURL(envKey string) (*url.URL, error) {
	raw := os.Getenv(envKey)
	if raw == "" {
		return nil, fmt.Errorf("proxy: %s is required", envKey)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() == "" || u.User == nil {
		return nil, fmt.Errorf("proxy: invalid %s", envKey)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("proxy: invalid %s", envKey)
	}
	if password, ok := u.User.Password(); !ok || password == "" || u.User.Username() == "" {
		return nil, fmt.Errorf("proxy: %s requires username and password", envKey)
	}
	return u, nil
}

func Transport(useProxy bool) (http.RoundTripper, error) {
	if !useProxy {
		return http.DefaultTransport, nil
	}
	u, err := proxyURL(envKey)
	if err != nil {
		return nil, err
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyURL(u)
	tr.OnProxyConnectResponse = func(_ context.Context, _ *url.URL, _ *http.Request, resp *http.Response) error {
		if zoneExhausted(resp) {
			return errZoneExhausted
		}
		return nil
	}
	if path := os.Getenv("BRIGHTDATA_CA_CERT"); path != "" {
		pem, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("proxy: read BRIGHTDATA_CA_CERT: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("proxy: system CAs: %w", err)
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("proxy: invalid BRIGHTDATA_CA_CERT")
		}
		tr.TLSClientConfig = &tls.Config{RootCAs: roots}
	}
	return tr, nil
}
