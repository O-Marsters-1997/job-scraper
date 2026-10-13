package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
)

func serveBody(method, contentType, body string) *httptest.ResponseRecorder {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	req := httptest.NewRequest(method, "/x", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	limitBody(8)(requireJSON(ok)).ServeHTTP(w, req)
	return w
}

func TestBodyMiddlewarePasses(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
	}{
		{"json body", http.MethodPost, "application/json; charset=utf-8", `{"a":1}`},
		{"bodyless post", http.MethodPost, "", ""},
		{"get is not checked", http.MethodGet, "text/plain", `x`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := serveBody(tt.method, tt.contentType, tt.body); w.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
			}
		})
	}
}

func TestBodyMiddlewareRejects(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		want        int
	}{
		{"over the cap", http.MethodPost, "application/json", `{"a":"too long"}`, http.StatusRequestEntityTooLarge},
		{"text body", http.MethodPut, "text/plain", `{"a":1}`, http.StatusUnsupportedMediaType},
		{"form body", http.MethodPatch, "application/x-www-form-urlencoded", `a=1`, http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveBody(tt.method, tt.contentType, tt.body)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.want, w.Body)
			}
			if got := handlerstest.DecodeJSON[map[string]any](t, w.Body.Bytes()); got["error"] == nil {
				t.Errorf("body %v lacks the error field", got)
			}
		})
	}
}
