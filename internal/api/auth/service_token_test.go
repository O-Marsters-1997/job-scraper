package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/auth"
)

func serveWithToken(t *testing.T, configured, header string) int {
	t.Helper()
	t.Setenv("INGEST_SERVICE_TOKEN", configured)
	h := auth.ServiceTokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/ingest", http.NoBody)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Code
}

func TestServiceTokenMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		header     string
		want       int
	}{
		{"matching bearer token passes", "s3cret", "Bearer s3cret", http.StatusOK},
		{"missing header is rejected", "s3cret", "", http.StatusUnauthorized},
		{"wrong token is rejected", "s3cret", "Bearer nope", http.StatusUnauthorized},
		{"unconfigured token rejects even an empty bearer", "", "Bearer ", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serveWithToken(t, tt.configured, tt.header); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}
