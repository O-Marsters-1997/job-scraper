// Adapter turns a service or provider method into an http.HandlerFunc: it
// resolves the caller, decodes the body, maps a returned apperr kind to a
// status, and encodes the result. See docs/adr/0020 and docs/adr/0021.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
)

// caller resolves the authenticated user's ID, writing a 401 and returning
// ok=false if no session is present.
func caller(w http.ResponseWriter, r *http.Request) (userID string, ok bool) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return "", false
	}
	return session.UserID, true
}

// writeJSON writes v as the response body with the given status. A nil slice
// is written as [], and a 204 status carries no body regardless of v.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		_, _ = w.Write([]byte("[]"))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// writeError maps err to a status via its apperr kind and writes
// {"error": msg, ...fields}. An error without a kind is logged with the
// route and returned as a 500 whose body hides the underlying message.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, ok := apperr.StatusFor(err)
	msg := err.Error()
	if !ok {
		slog.Error("unhandled handler error",
			slog.String("route", r.Method+" "+r.URL.Path),
			slog.Any("err", err),
		)
		status = http.StatusInternalServerError
		msg = "internal server error"
	}
	body := map[string]any{"error": msg}
	for k, v := range apperr.FieldsFor(err) {
		body[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
