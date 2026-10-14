package proxy

import (
	"io"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	routeResidential = "residential"
	routeDirect      = "direct"

	directionRead  = "read"
	directionWrite = "write"
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
	FetchWireBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "jobscraper_fetch_wire_bytes_total",
		Help: "Bytes read from and written to the proxy connection, CONNECT and TLS included, by source, route and direction.",
	}, []string{"source", "route", "direction"})
)

func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(FetchRequests, FetchBytes, FetchWireBytes)
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

type countingConn struct {
	net.Conn
	read, written prometheus.Counter
}

func newCountingConn(c net.Conn, source, route string) *countingConn {
	return &countingConn{
		Conn:    c,
		read:    FetchWireBytes.WithLabelValues(source, route, directionRead),
		written: FetchWireBytes.WithLabelValues(source, route, directionWrite),
	}
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.read.Add(float64(n))
	return n, err
}

func (c *countingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	c.written.Add(float64(n))
	return n, err
}

func record(req *http.Request, source, route string, resp *http.Response, err error) {
	FetchRequests.WithLabelValues(source, route, outcome(req, resp, err)).Inc()
	if err == nil {
		resp.Body = &countingBody{ReadCloser: resp.Body, bytes: FetchBytes.WithLabelValues(source, route)}
	}
}
