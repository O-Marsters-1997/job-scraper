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
		wantErr     bool
	}{
		{name: "direct returns default transport", useProxy: false, wantDefault: true},
		{name: "proxy on with valid URL uses proxy transport", useProxy: true, envVal: "http://user:pass@brd.superproxy.io:33335"},
		{name: "proxy on without env var fails closed", useProxy: true, envVal: "", wantErr: true},
		{name: "proxy on with invalid URL returns error", useProxy: true, envVal: "://bad-url", wantErr: true},
		{name: "proxy on without credentials returns error", useProxy: true, envVal: "http://brd.superproxy.io:33335", wantErr: true},
		{name: "proxy on without port returns error", useProxy: true, envVal: "http://user:pass@brd.superproxy.io", wantErr: true},
		{name: "proxy on with path returns error", useProxy: true, envVal: "http://user:pass@brd.superproxy.io:33335/path", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envKey, tt.envVal)
			t.Setenv("BRIGHTDATA_CA_CERT", "")

			tr, err := Transport(tt.useProxy)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
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
