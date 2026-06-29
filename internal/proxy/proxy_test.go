package proxy

import (
	"net/http"
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
		{name: "proxy on without env var falls back to default", useProxy: true, envVal: "", wantDefault: true},
		{name: "proxy on with invalid URL returns error", useProxy: true, envVal: "://bad-url", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envKey, tt.envVal)

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
