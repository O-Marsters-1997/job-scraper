package proxy_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

const proxyEnv = "BRIGHTDATA_PROXY_URL"

func TestTransport(t *testing.T) {
	t.Run("direct returns default transport", func(t *testing.T) {
		tr, err := proxy.Transport(false)
		if err != nil {
			t.Fatal(err)
		}
		if tr != http.DefaultTransport {
			t.Fatal("expected http.DefaultTransport")
		}
	})

	t.Run("proxy on with valid URL uses proxy transport", func(t *testing.T) {
		t.Setenv(proxyEnv, "http://user:pass@brd.superproxy.io:33335")
		t.Setenv("BRIGHTDATA_CA_CERT", "")
		tr, err := proxy.Transport(true)
		if err != nil {
			t.Fatal(err)
		}
		if tr == http.DefaultTransport {
			t.Fatal("expected proxy transport, got DefaultTransport")
		}
	})

	for name, env := range map[string]string{
		"proxy on without env var fails closed":      "",
		"proxy on with invalid URL returns error":    "://bad-url",
		"proxy on without credentials returns error": "http://brd.superproxy.io:33335",
		"proxy on without port returns error":        "http://user:pass@brd.superproxy.io",
		"proxy on with path returns error":           "http://user:pass@brd.superproxy.io:33335/path",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(proxyEnv, env)
			t.Setenv("BRIGHTDATA_CA_CERT", "")
			if _, err := proxy.Transport(true); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestProxyConnectClassifiesZoneExhaustion(t *testing.T) {
	t.Setenv(proxyEnv, "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "")
	tr, err := proxy.Transport(true)
	if err != nil {
		t.Fatal(err)
	}
	resp := &http.Response{StatusCode: 502, Header: http.Header{"X-Brd-Err-Code": []string{"client_10100"}}}
	if err := tr.(*http.Transport).OnProxyConnectResponse(context.Background(), &url.URL{}, &http.Request{}, resp); !proxy.IsZonePaused(err) {
		t.Fatalf("CONNECT exhaustion = %v", err)
	}
}

func TestValidateRejectsUnreadableCA(t *testing.T) {
	t.Setenv(proxyEnv, "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "/does-not-exist")
	if err := proxy.Validate(); err == nil {
		t.Fatal("expected CA startup validation error")
	}
}
