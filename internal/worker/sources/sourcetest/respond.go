package sourcetest

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Responder is a transport that answers every request with a fixed status and body.
type Responder struct {
	status int
	body   string
	Last   *http.Request
}

// RespondStatus returns a Responder serving body with the given status code.
func RespondStatus(status int, body string) *Responder { return &Responder{status: status, body: body} }

// Respond returns a Responder serving body; Last holds the most recent request.
func Respond(body string) *Responder { return &Responder{status: http.StatusOK, body: body} }

func (r *Responder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.Last = req
	return &http.Response{StatusCode: r.status, Status: fmt.Sprintf("%d %s", r.status, http.StatusText(r.status)), Body: io.NopCloser(strings.NewReader(r.body))}, nil
}

// PathResponder serves a fixed body per request path and 404s any other path. It is safe for
// concurrent use.
type PathResponder struct {
	bodies map[string]string
	mu     sync.Mutex
	calls  []string
}

// RespondByPath returns a PathResponder. A key matches a request's path and query
// ("/v1/postings?offset=2") before its bare path ("/v1/postings").
func RespondByPath(bodies map[string]string) *PathResponder { return &PathResponder{bodies: bodies} }

func (r *PathResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.calls = append(r.calls, req.URL.Path)
	r.mu.Unlock()

	body, ok := r.bodies[req.URL.RequestURI()]
	if !ok {
		body, ok = r.bodies[req.URL.Path]
	}
	if !ok {
		return RespondStatus(http.StatusNotFound, "").RoundTrip(req)
	}
	return Respond(body).RoundTrip(req)
}

// Calls returns how many requests have been made to path.
func (r *PathResponder) Calls(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.calls {
		if p == path {
			n++
		}
	}
	return n
}
