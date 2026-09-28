package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
)

func TestTransport(t *testing.T) {
	tests := []struct {
		name        string
		useProxy    bool
		envVal      string
		wantDefault bool
	}{
		{name: "direct returns default transport", useProxy: false, wantDefault: true},
		{name: "proxy on with valid URL uses proxy transport", useProxy: true, envVal: "http://user:pass@brd.superproxy.io:33335"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envKey, tt.envVal)
			t.Setenv("BRIGHTDATA_CA_CERT", "")

			tr, err := Transport(tt.useProxy)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantDefault && tr != http.DefaultTransport {
				t.Fatal("expected http.DefaultTransport")
			}
			if !tt.wantDefault && tr == http.DefaultTransport {
				t.Fatal("expected proxy transport, got DefaultTransport")
			}
		})
	}
}

func TestTransport_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
	}{
		{name: "proxy on without env var fails closed", envVal: ""},
		{name: "proxy on with invalid URL returns error", envVal: "://bad-url"},
		{name: "proxy on without credentials returns error", envVal: "http://brd.superproxy.io:33335"},
		{name: "proxy on without port returns error", envVal: "http://user:pass@brd.superproxy.io"},
		{name: "proxy on with path returns error", envVal: "http://user:pass@brd.superproxy.io:33335/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envKey, tt.envVal)
			t.Setenv("BRIGHTDATA_CA_CERT", "")

			if _, err := Transport(true); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestProxyConnectClassifiesZoneExhaustion(t *testing.T) {
	t.Setenv(envKey, "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "")
	tr, err := Transport(true)
	if err != nil {
		t.Fatal(err)
	}
	resp := &http.Response{StatusCode: 502, Header: http.Header{"X-Brd-Err-Code": []string{"client_10100"}}}
	if err := tr.(*http.Transport).OnProxyConnectResponse(context.Background(), &url.URL{}, &http.Request{}, resp); !errors.Is(err, errZoneExhausted) {
		t.Fatalf("CONNECT exhaustion = %v", err)
	}
}

func TestValidateRejectsUnreadableCA(t *testing.T) {
	t.Setenv(envKey, "http://user:pass@brd.superproxy.io:33335")
	t.Setenv("BRIGHTDATA_CA_CERT", "/does-not-exist")
	if err := Validate(); err == nil {
		t.Fatal("expected CA startup validation error")
	}
}
