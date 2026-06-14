package proxy

import (
	"net/http"
	"testing"
)

func TestTransport(t *testing.T) {
	tests := []struct {
		name        string
		tier        Tier
		envKey      string
		envVal      string
		wantDefault bool
		wantErr     bool
	}{
		{name: "Direct returns default transport", tier: Direct, wantDefault: true},
		{name: "Datacenter with valid URL uses proxy", tier: Datacenter, envKey: "PROXY_DATACENTER_URL", envVal: "http://proxy.example.com:8080"},
		{name: "Datacenter without env var falls back to default", tier: Datacenter, envKey: "PROXY_DATACENTER_URL", envVal: "", wantDefault: true},
		{name: "Residential with valid URL uses proxy", tier: Residential, envKey: "PROXY_RESIDENTIAL_URL", envVal: "http://residential.example.com:9090"},
		{name: "Residential without env var falls back to default", tier: Residential, envKey: "PROXY_RESIDENTIAL_URL", envVal: "", wantDefault: true},
		{name: "unknown tier returns error", tier: Tier(99), wantErr: true},
		{name: "Datacenter with invalid URL returns error", tier: Datacenter, envKey: "PROXY_DATACENTER_URL", envVal: "://bad-url", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			tr, err := Transport(tt.tier)

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
