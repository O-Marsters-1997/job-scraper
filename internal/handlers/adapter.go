// Adapter turns a service or provider method into an http.HandlerFunc: it
// resolves the caller, decodes the body, maps a returned apperr kind to a
// status, and encodes the result. See docs/adr/0020.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
)

// Caller resolves the authenticated user's ID, writing a 401 and returning
// ok=false if no session is present.
func Caller(w http.ResponseWriter, r *http.Request) (userID string, ok bool) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		auth.WriteUnauthorized(w)
		return "", false
	}
	return session.UserID, true
}

// DecodeJSON decodes the request body into T, writing a 400 and returning
// ok=false on failure.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (in T, ok bool) {
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, r, apperr.Invalid("bad request"))
		var zero T
		return zero, false
	}
	return in, true
}

// WriteJSON writes v as the response body with the given status. A nil slice
// is written as [], and a 204 status carries no body regardless of v.
func WriteJSON(w http.ResponseWriter, status int, v any) {
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

// WriteError maps err to a status via its apperr kind and writes
// {"error": msg}. An error without a kind is logged with the route and
// returned as a 500 whose body hides the underlying message.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func User[Out any](fn func(ctx context.Context, userID string) (Out, error), status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := Caller(w, r)
		if !ok {
			return
		}
		out, err := fn(r.Context(), userID)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, status, out)
	}
}

// ID adapts a function of (ctx, userID, id) to an http.HandlerFunc, taking id
// from the "id" chi URL param.
func ID[Out any](fn func(ctx context.Context, userID, id string) (Out, error), status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := Caller(w, r)
		if !ok {
			return
		}
		out, err := fn(r.Context(), userID, chi.URLParam(r, "id"))
		if err != nil {
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, status, out)
	}
}

// Body adapts a function of (ctx, userID, in) to an http.HandlerFunc, in
// decoded from the request body.
func Body[In, Out any](fn func(ctx context.Context, userID string, in In) (Out, error), status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := Caller(w, r)
		if !ok {
			return
		}
		in, ok := DecodeJSON[In](w, r)
		if !ok {
			return
		}
		out, err := fn(r.Context(), userID, in)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, status, out)
	}
}

// BodyID adapts a function of (ctx, userID, id, in) to an http.HandlerFunc,
// combining ID and Body.
func BodyID[In, Out any](fn func(ctx context.Context, userID, id string, in In) (Out, error), status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := Caller(w, r)
		if !ok {
			return
		}
		in, ok := DecodeJSON[In](w, r)
		if !ok {
			return
		}
		out, err := fn(r.Context(), userID, chi.URLParam(r, "id"), in)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, status, out)
	}
}
