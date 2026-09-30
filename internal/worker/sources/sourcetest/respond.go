package sourcetest

import (
	"io"
	"net/http"
	"strings"
)

// Responder is a transport that answers every request with 200 and a fixed body.
type Responder struct {
	body string
	Last *http.Request
}

// Respond returns a Responder serving body; Last holds the most recent request.
func Respond(body string) *Responder { return &Responder{body: body} }

func (r *Responder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.Last = req
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(r.body))}, nil
}
