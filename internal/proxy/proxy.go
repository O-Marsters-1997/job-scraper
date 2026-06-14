package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// Tier classifies the proxy behaviour for outbound HTTP requests.
type Tier int

const (
	// Direct sends requests without a proxy.
	Direct Tier = iota
	// Datacenter routes through a datacenter proxy; reads PROXY_DATACENTER_URL.
	Datacenter
	// Residential routes through a residential proxy; reads PROXY_RESIDENTIAL_URL.
	Residential
)

// Transport returns an http.RoundTripper for tier.
// Datacenter reads PROXY_DATACENTER_URL; Residential reads PROXY_RESIDENTIAL_URL.
// If the env var is unset the tier falls back to http.DefaultTransport.
func Transport(tier Tier) (http.RoundTripper, error) {
	switch tier {
	case Direct:
		return http.DefaultTransport, nil
	case Datacenter:
		return fromEnv("PROXY_DATACENTER_URL")
	case Residential:
		return fromEnv("PROXY_RESIDENTIAL_URL")
	default:
		return nil, fmt.Errorf("proxy: unknown tier %d", tier)
	}
}

func fromEnv(key string) (http.RoundTripper, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return http.DefaultTransport, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("proxy: parse %s: %w", key, err)
	}
	return &http.Transport{Proxy: http.ProxyURL(u)}, nil
}
