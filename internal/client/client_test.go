package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/client"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header string
		want   client.Kind
	}{
		{"ios header", "ios", client.IOS},
		{"web header", "web", client.Web},
		{"missing header", "", client.Web},
		{"unknown value", "android", client.Web},
		{"wrong case", "IOS", client.Web},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got client.Kind
			h := client.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = client.FromContext(r.Context())
			}))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set(client.Header, tt.header)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tt.want {
				t.Errorf("FromContext with %s=%q = %q, want %q", client.Header, tt.header, got, tt.want)
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	t.Parallel()
	if got := client.FromContext(t.Context()); got != client.Web {
		t.Errorf("FromContext(no middleware) = %q, want %q", got, client.Web)
	}
}
