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

func decodeBody[T any](w http.ResponseWriter, r *http.Request) (in T, ok bool) {
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, r, apperr.Invalid("bad request"))
		var zero T
		return zero, false
	}
	return in, true
}

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

func fillPath(r *http.Request, in any) {
	v := reflect.ValueOf(in).Elem()
	t := v.Type()
	for i := range t.NumField() {
		if tag := t.Field(i).Tag.Get("path"); tag != "" {
			v.Field(i).SetString(chi.URLParam(r, tag))
		}
	}
}

func respond[Out any](w http.ResponseWriter, status int, out Out) {
	if _, void := any(out).(struct{}); void {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	writeJSON(w, status, out)
}

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
