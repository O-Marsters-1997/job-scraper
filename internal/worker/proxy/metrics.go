package proxy

import (
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	routeResidential = "residential"
	routeDirect      = "direct"
)

var (
	FetchRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "jobscraper_fetch_requests_total",
		Help: "Outbound fetch attempts by source, route and outcome (ok, blocked, gone, error).",
	}, []string{"source", "route", "outcome"})
	FetchBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "jobscraper_fetch_bytes_total",
		Help: "Response body bytes read by source and route.",
	}, []string{"source", "route"})
)

func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(FetchRequests, FetchBytes)
}

func outcome(req *http.Request, resp *http.Response, err error) string {
	switch {
	case err != nil:
		return "error"
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return "gone"
	case blocked(req, resp):
		return "blocked"
	case resp.StatusCode >= 400:
		return "error"
	}
	return "ok"
}

type countingBody struct {
	io.ReadCloser
	bytes prometheus.Counter
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.bytes.Add(float64(n))
	return n, err
}

func record(req *http.Request, source, route string, resp *http.Response, err error) {
	FetchRequests.WithLabelValues(source, route, outcome(req, resp, err)).Inc()
	if err == nil {
		resp.Body = &countingBody{ReadCloser: resp.Body, bytes: FetchBytes.WithLabelValues(source, route)}
	}
}
