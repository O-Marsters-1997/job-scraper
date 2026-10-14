package proxy

import (
	"net/http"
	"time"
)

func WithRetryBackoff(tr http.RoundTripper, d time.Duration) {
	rt, ok := tr.(*residentialTransport)
	if !ok {
		panic("WithRetryBackoff needs the residential transport")
	}
	rt.backoff = d
}
