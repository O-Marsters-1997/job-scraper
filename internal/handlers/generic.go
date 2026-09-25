// Generic CRUD wrappers over a service function. Each one fixes a function
// shape and a success status; a returned Out of struct{} always means 204
// regardless of the verb's default. Path IDs travel on the input dto via a
// `path:"..."` struct tag, filled from chi URL params before the service is
// called, so the request body can never set them. See docs/adr/0021.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

// decodeBody decodes the request body into T. An empty body decodes to T's
// zero value rather than failing: bodyless actions (e.g. hide/show) and
// dtos with only path-tagged fields never need to send one, and a required
// field's absence is a service-level validation error, not a decode error.
func decodeBody[T any](w http.ResponseWriter, r *http.Request) (in T, ok bool) {
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, r, apperr.Invalid("bad request"))
		var zero T
		return zero, false
	}
	return in, true
}

// decodeQuery flattens the request's URL query into Q by JSON round-trip: Q
// should declare string fields with json tags matching the query keys, and
// the service parses and validates them.
func decodeQuery[Q any](r *http.Request) (Q, error) {
	var q Q
	values := r.URL.Query()
	flat := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			flat[k] = v[0]
		}
	}
	b, err := json.Marshal(flat)
	if err != nil {
		return q, err
	}
	if err := json.Unmarshal(b, &q); err != nil {
		return q, err
	}
	return q, nil
}

// fillPath sets every `path:"name"` tagged field on in from the matching
// chi URL param.
func fillPath(r *http.Request, in any) {
	v := reflect.ValueOf(in).Elem()
	t := v.Type()
	for i := range t.NumField() {
		if tag := t.Field(i).Tag.Get("path"); tag != "" {
			v.Field(i).SetString(chi.URLParam(r, tag))
		}
	}
}

// respond writes out with status, except a struct{} Out always writes 204
// with no body.
func respond[Out any](w http.ResponseWriter, status int, out Out) {
	if _, void := any(out).(struct{}); void {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	writeJSON(w, status, out)
}

// GetAll adapts (ctx, userID) -> (Out, error) to a 200 collection or
// singleton read.
func GetAll[Out any](fn func(ctx context.Context, userID string) (Out, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		out, err := fn(r.Context(), userID)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// GetByID adapts (ctx, userID, id) -> (Out, error) to a 200 read, id taken
// from the "id" chi URL param. Reused for any action whose shape matches,
// regardless of HTTP method (e.g. POST .../scrape).
func GetByID[Out any](fn func(ctx context.Context, userID, id string) (Out, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		out, err := fn(r.Context(), userID, chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// Query adapts (ctx, userID, q Q) -> (Out, error) to a 200 read, q decoded
// from the URL query string.
func Query[Q, Out any](fn func(ctx context.Context, userID string, q Q) (Out, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		q, err := decodeQuery[Q](r)
		if err != nil {
			writeError(w, r, apperr.Invalid("bad request"))
			return
		}
		out, err := fn(r.Context(), userID, q)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// Create adapts (ctx, userID, in In) -> (Out, error) to a 201 create; a
// struct{} Out writes 204 instead. Path-tagged fields on In are filled from
// chi URL params after decoding.
func Create[In, Out any](fn func(ctx context.Context, userID string, in In) (Out, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		in, ok := decodeBody[In](w, r)
		if !ok {
			return
		}
		fillPath(r, &in)
		out, err := fn(r.Context(), userID, in)
		if err != nil {
			writeError(w, r, err)
			return
		}
		respond(w, http.StatusCreated, out)
	}
}

// Update adapts (ctx, userID, in In) -> (Out, error) to a 200 update; a
// struct{} Out writes 204 instead. Path-tagged fields on In are filled from
// chi URL params after decoding, covering both a single {id} and a
// multi-segment path (e.g. {docId}/{tabId}).
func Update[In, Out any](fn func(ctx context.Context, userID string, in In) (Out, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		in, ok := decodeBody[In](w, r)
		if !ok {
			return
		}
		fillPath(r, &in)
		out, err := fn(r.Context(), userID, in)
		if err != nil {
			writeError(w, r, err)
			return
		}
		respond(w, http.StatusOK, out)
	}
}

// Delete adapts (ctx, userID, id) -> error to a 204 delete, id taken from
// the "id" chi URL param.
func Delete(fn func(ctx context.Context, userID, id string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		if err := fn(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
			writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
