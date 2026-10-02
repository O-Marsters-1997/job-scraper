package sourcetest

import (
	"fmt"
	"io"
	"net/http"
	"strings"
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
