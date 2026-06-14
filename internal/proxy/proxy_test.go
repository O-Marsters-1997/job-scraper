package proxy

import (
	"net/http"
	"testing"
)

func TestTransport(t *testing.T) {
	t.Run("Direct returns default transport", func(t *testing.T) {
		tr, err := Transport(Direct)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr != http.DefaultTransport {
			t.Fatal("expected http.DefaultTransport")
		}
	})

	t.Run("Datacenter with env var returns proxy transport", func(t *testing.T) {
		t.Setenv("PROXY_DATACENTER_URL", "http://proxy.example.com:8080")
		tr, err := Transport(Datacenter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr == http.DefaultTransport {
			t.Fatal("expected a proxy transport, got DefaultTransport")
		}
	})

	t.Run("Datacenter without env var falls back to default transport", func(t *testing.T) {
		t.Setenv("PROXY_DATACENTER_URL", "")
		tr, err := Transport(Datacenter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr != http.DefaultTransport {
			t.Fatal("expected http.DefaultTransport when env var is unset")
		}
	})

	t.Run("Residential with env var returns proxy transport", func(t *testing.T) {
		t.Setenv("PROXY_RESIDENTIAL_URL", "http://residential.example.com:9090")
		tr, err := Transport(Residential)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr == http.DefaultTransport {
			t.Fatal("expected a proxy transport, got DefaultTransport")
		}
	})

	t.Run("Residential without env var falls back to default transport", func(t *testing.T) {
		t.Setenv("PROXY_RESIDENTIAL_URL", "")
		tr, err := Transport(Residential)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr != http.DefaultTransport {
			t.Fatal("expected http.DefaultTransport when env var is unset")
		}
	})

	t.Run("Unknown tier returns error", func(t *testing.T) {
		_, err := Transport(Tier(99))
		if err == nil {
			t.Fatal("expected error for unknown tier")
		}
	})

	t.Run("Datacenter with invalid URL returns error", func(t *testing.T) {
		t.Setenv("PROXY_DATACENTER_URL", "://bad-url")
		_, err := Transport(Datacenter)
		if err == nil {
			t.Fatal("expected error for invalid proxy URL")
		}
	})
}
