package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/logger"
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

// DecodeBody decodes the request body as JSON, treating an empty body as
// the zero value.
func DecodeBody[T any](r *http.Request) (T, error) {
	var in T
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		return in, apperr.Invalid("bad request")
	}
	return in, nil
}

// DecodeQuery decodes the request's URL query string into Q, matching query
// keys to Q's JSON field names.
func DecodeQuery[Q any](r *http.Request) (Q, error) {
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

func fillPath(r *http.Request, in any) {
	v := reflect.ValueOf(in).Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := range t.NumField() {
		if tag := t.Field(i).Tag.Get("path"); tag != "" {
			v.Field(i).SetString(chi.URLParam(r, tag))
		}
	}
}

func respond[Out any](w http.ResponseWriter, status int, out Out) {
	if _, void := any(out).(struct{}); void {
		WriteJSON(w, http.StatusNoContent, nil)
		return
	}
	WriteJSON(w, status, out)
}

// WriteJSON writes v as a JSON response body with status, writing a nil
// slice as "[]" and skipping the body entirely for a 204.
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

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, ok := apperr.StatusFor(err)
	msg := err.Error()
	if !ok {
		slog.ErrorContext(r.Context(), "unhandled handler error",
			slog.String(logger.KeyMethod, r.Method),
			slog.String(logger.KeyRoute, chi.RouteContext(r.Context()).RoutePattern()),
			slog.Any(logger.KeyErr, err),
		)
		status = http.StatusInternalServerError
		msg = "internal server error"
	}
	body := map[string]any{"error": msg}
	maps.Copy(body, apperr.FieldsFor(err))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type byID struct{ userID, id string }

func decodeByID(r *http.Request) (byID, error) {
	uid, err := UserID(r)
	if err != nil {
		return byID{}, err
	}
	return byID{userID: uid, id: chi.URLParam(r, "id")}, nil
}

// GetAll adapts (ctx, userID) -> (Out, error) to a 200 collection or
// singleton read.
func GetAll[Out any](fn func(ctx context.Context, userID string) (Out, error)) http.HandlerFunc {
	return Handle(UserID, fn, respondJSON[Out](http.StatusOK))
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
// from the URL query string. Path-tagged fields on Q are filled from chi URL
// params.
func Query[Q, Out any](fn func(ctx context.Context, userID string, q Q) (Out, error)) http.HandlerFunc {
	type req struct {
		userID string
		q      Q
	}
	return Handle(
		func(r *http.Request) (req, error) {
			uid, err := UserID(r)
			if err != nil {
				return req{}, err
			}
			q, err := DecodeQuery[Q](r)
			if err != nil {
				return req{}, apperr.Invalid("bad request")
			}
			fillPath(r, &q)
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
	uid, err := UserID(r)
	if err != nil {
		return userInput[In]{}, err
	}
	in, err := DecodeBody[In](r)
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
// chi URL params after decoding (a single {id} or a multi-segment path).
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
