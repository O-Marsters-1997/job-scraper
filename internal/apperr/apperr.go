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
	var ae *Error
	if errors.As(err, &ae) {
		return ae.kind.Status(), true
	}
	return 0, false
}
