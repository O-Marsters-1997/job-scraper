// Package apperr defines error kinds with fixed HTTP statuses, so an error
// carries its mapping from where it originates and the HTTP adapter never
// needs to know about a specific error type.
package apperr

import (
	"errors"
	"net/http"
)

type Kind int

const (
	KindInvalid Kind = iota
	KindUnauthorized
	KindNotFound
	KindConflict
	KindUnprocessable
	KindUpstream
	KindUnavailable
)

func (k Kind) Status() int {
	switch k {
	case KindInvalid:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnprocessable:
		return http.StatusUnprocessableEntity
	case KindUpstream:
		return http.StatusBadGateway
	case KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Error is a sentinel with a kind. Compare it with errors.Is against the
// package variable that declared it; detect it through a wrapper chain with
// errors.As, which StatusFor already does.
type Error struct {
	kind Kind
	msg  string
}

func (e *Error) Error() string { return e.msg }
func (e *Error) Kind() Kind    { return e.kind }

func Invalid(msg string) error       { return &Error{kind: KindInvalid, msg: msg} }
func Unauthorized(msg string) error  { return &Error{kind: KindUnauthorized, msg: msg} }
func NotFound(msg string) error      { return &Error{kind: KindNotFound, msg: msg} }
func Conflict(msg string) error      { return &Error{kind: KindConflict, msg: msg} }
func Unprocessable(msg string) error { return &Error{kind: KindUnprocessable, msg: msg} }
func Upstream(msg string) error      { return &Error{kind: KindUpstream, msg: msg} }
func Unavailable(msg string) error   { return &Error{kind: KindUnavailable, msg: msg} }

// StatusFor reports the HTTP status for err's kind, found via errors.As so a
// wrapped apperr.Error is still detected. ok is false when err carries no kind.
func StatusFor(err error) (status int, ok bool) {
	ae, ok := errors.AsType[*Error](err)
	if !ok {
		return 0, false
	}
	return ae.kind.Status(), true
}

// fielded wraps an error with extra fields the adapter merges into the JSON
// error body next to "error", e.g. the in-use count on a 409.
type fielded struct {
	error
	fields map[string]any
}

func (f *fielded) Unwrap() error { return f.error }

// WithFields attaches extra fields to err for the adapter to include in the
// response body. err should carry a kind (e.g. from Conflict) so it still
// maps to a status.
func WithFields(err error, fields map[string]any) error {
	return &fielded{error: err, fields: fields}
}

// FieldsFor returns the fields attached via WithFields, found through the
// wrap chain, or nil if none were attached.
func FieldsFor(err error) map[string]any {
	f, ok := errors.AsType[*fielded](err)
	if !ok {
		return nil
	}
	return f.fields
}
