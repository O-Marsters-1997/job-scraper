// Package handlerstest holds the generic route-level checks every module's
// routes_test.go runs against its own router (ADR 0012): auth, malformed
// bodies and unknown path IDs, so no module writes them itself.
package handlerstest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

const testUserID = "handlerstest-user"

func splitRoute(t *testing.T, route string) (method, path string) {
	t.Helper()
	method, path, ok := strings.Cut(route, " ")
	if !ok {
		t.Fatalf("route %q must be \"METHOD /path\"", route)
	}
	return method, path
}

func fillPathID(path, id string) string {
	return strings.ReplaceAll(path, "{id}", id)
}

func authedRequest(method, path string, body *strings.Reader) *http.Request {
	var r io.Reader = http.NoBody
	if body != nil {
		r = body
	}
	return Authed(httptest.NewRequest(method, path, r))
}

const UserID = testUserID

func Authed(req *http.Request) *http.Request {
	return req.WithContext(handlers.WithSession(req.Context(), dto.Session{UserID: testUserID}))
}

// RequiresAuth asserts each route responds 401 when no session is attached
// to the request.
func RequiresAuth(t *testing.T, h http.Handler, routes ...string) {
	t.Helper()
	for _, route := range routes {
		method, path := splitRoute(t, route)
		req := httptest.NewRequest(method, fillPathID(path, "test-id"), http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s without a session: status = %d, want %d", route, w.Code, http.StatusUnauthorized)
		}
	}
}

// RejectsMalformedBody asserts each route responds 400 to a body that isn't
// valid JSON, with a session already attached so auth doesn't mask the check.
func RejectsMalformedBody(t *testing.T, h http.Handler, routes ...string) {
	t.Helper()
	for _, route := range routes {
		method, path := splitRoute(t, route)
		req := authedRequest(method, fillPathID(path, "test-id"), strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s with a malformed body: status = %d, want %d", route, w.Code, http.StatusBadRequest)
		}
	}
}

// RejectsBadPathID asserts each route responds 404 for an {id} path segment
// that doesn't exist, with a session already attached so auth doesn't mask
// the check.
func RejectsBadPathID(t *testing.T, h http.Handler, routes ...string) {
	t.Helper()
	for _, route := range routes {
		method, path := splitRoute(t, route)
		req := authedRequest(method, fillPathID(path, "does-not-exist"), nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s with an unknown id: status = %d, want %d", route, w.Code, http.StatusNotFound)
		}
	}
}
