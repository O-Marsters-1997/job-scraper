package logger

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HeaderRunID carries a run's ID across the worker's calls to the api's
// /ingest routes (ADR 0013).
const HeaderRunID = "X-Run-ID"

// Middleware puts request_id, a fresh trace_id, and run_id from HeaderRunID
// when present, on the request's context. Mount it after chi's
// middleware.RequestID.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := With(r.Context(),
			slog.String(KeyRequestID, middleware.GetReqID(r.Context())),
			slog.String(KeyTraceID, uuid.NewString()),
		)
		if runID := r.Header.Get(HeaderRunID); runID != "" {
			ctx = With(ctx, slog.String(KeyRunID, runID))
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Transport sets HeaderRunID from the request context's run_id, when With
// put one there. Wrap only a client that calls our own api with it: any
// other client would leak the header to a third party.
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(runIDTransport{base})
}

type runIDTransport struct {
	base http.RoundTripper
}

func (t runIDTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if runID, ok := ctxAttr(req.Context(), KeyRunID); ok {
		req = req.Clone(req.Context())
		req.Header.Set(HeaderRunID, runID.String())
	}
	return t.base.RoundTrip(req)
}
