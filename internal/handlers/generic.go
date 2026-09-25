// Handle is the pipeline every handler in this package is built from:
// decode, call, respond. See docs/adr/0022.
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

// Handle wires decode -> call -> respond into an http.HandlerFunc. A decode
// or call error goes to writeError; respond only runs on success and can't
// fail itself — anything that can fail belongs in decode or call.
func Handle[Req, Res any](
	decode func(r *http.Request) (Req, error),
	call func(ctx context.Context, req Req) (Res, error),
	respond func(w http.ResponseWriter, r *http.Request, res Res),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decode(r)
		if err != nil {
			writeError(w, r, err)
			return
		}
		res, err := call(r.Context(), req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		respond(w, r, res)
	}
}

func respondJSON[Res any](status int) func(http.ResponseWriter, *http.Request, Res) {
	return func(w http.ResponseWriter, _ *http.Request, res Res) {
		respond(w, status, res)
	}
}

func pass[T any](_ context.Context, v T) (T, error) { return v, nil }

// decodeBody decodes the request body into T. An empty body decodes to T's
// zero value rather than failing: bodyless actions (e.g. hide/show) and
// dtos with only path-tagged fields never need to send one, and a required
// field's absence is a service-level validation error, not a decode error.
func decodeBody[T any](r *http.Request) (T, error) {
	var in T
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		return in, apperr.Invalid("bad request")
	}
	return in, nil
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

type byID struct{ userID, id string }

func decodeByID(r *http.Request) (byID, error) {
	uid, err := userID(r)
	if err != nil {
		return byID{}, err
	}
	return byID{userID: uid, id: chi.URLParam(r, "id")}, nil
}

// GetAll adapts (ctx, userID) -> (Out, error) to a 200 collection or
// singleton read.
func GetAll[Out any](fn func(ctx context.Context, userID string) (Out, error)) http.HandlerFunc {
	return Handle(userID, fn, respondJSON[Out](http.StatusOK))
}

// GetByID adapts (ctx, userID, id) -> (Out, error) to a 200 read, id taken
// from the "id" chi URL param. Reused for any action whose shape matches,
// regardless of HTTP method (e.g. POST .../scrape).
func GetByID[Out any](fn func(ctx context.Context, userID, id string) (Out, error)) http.HandlerFunc {
	return Handle(
		decodeByID,
		func(ctx context.Context, in byID) (Out, error) { return fn(ctx, in.userID, in.id) },
		respondJSON[Out](http.StatusOK),
	)
}

// Query adapts (ctx, userID, q Q) -> (Out, error) to a 200 read, q decoded
// from the URL query string.
func Query[Q, Out any](fn func(ctx context.Context, userID string, q Q) (Out, error)) http.HandlerFunc {
	type req struct {
		userID string
		q      Q
	}
	return Handle(
		func(r *http.Request) (req, error) {
			uid, err := userID(r)
			if err != nil {
				return req{}, err
			}
			q, err := decodeQuery[Q](r)
			if err != nil {
				return req{}, apperr.Invalid("bad request")
			}
			return req{userID: uid, q: q}, nil
		},
		func(ctx context.Context, in req) (Out, error) { return fn(ctx, in.userID, in.q) },
		respondJSON[Out](http.StatusOK),
	)
}

type userInput[In any] struct {
	userID string
	in     In
}

func decodeUserInput[In any](r *http.Request) (userInput[In], error) {
	uid, err := userID(r)
	if err != nil {
		return userInput[In]{}, err
	}
	in, err := decodeBody[In](r)
	if err != nil {
		return userInput[In]{}, err
	}
	fillPath(r, &in)
	return userInput[In]{userID: uid, in: in}, nil
}

func toServiceCall[In, Out any](fn func(ctx context.Context, userID string, in In) (Out, error)) func(context.Context, userInput[In]) (Out, error) {
	return func(ctx context.Context, req userInput[In]) (Out, error) { return fn(ctx, req.userID, req.in) }
}

// Create adapts (ctx, userID, in In) -> (Out, error) to a 201 create; a
// struct{} Out writes 204 instead. Path-tagged fields on In are filled from
// chi URL params after decoding.
func Create[In, Out any](fn func(ctx context.Context, userID string, in In) (Out, error)) http.HandlerFunc {
	return Handle(decodeUserInput[In], toServiceCall(fn), respondJSON[Out](http.StatusCreated))
}

// Update adapts (ctx, userID, in In) -> (Out, error) to a 200 update; a
// struct{} Out writes 204 instead. Path-tagged fields on In are filled from
// chi URL params after decoding, covering both a single {id} and a
// multi-segment path (e.g. {docId}/{tabId}).
func Update[In, Out any](fn func(ctx context.Context, userID string, in In) (Out, error)) http.HandlerFunc {
	return Handle(decodeUserInput[In], toServiceCall(fn), respondJSON[Out](http.StatusOK))
}

// Delete adapts (ctx, userID, id) -> error to a 204 delete, id taken from
// the "id" chi URL param.
func Delete(fn func(ctx context.Context, userID, id string) error) http.HandlerFunc {
	return Handle(
		decodeByID,
		func(ctx context.Context, in byID) (struct{}, error) { return struct{}{}, fn(ctx, in.userID, in.id) },
		respondJSON[struct{}](http.StatusNoContent),
	)
}
