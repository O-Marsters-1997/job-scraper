package jev

import "time"

// Error classifies a Jev call failure for the caller's retry scheduling.
type Error struct {
	Terminal   bool
	RetryAfter time.Duration
	err        error
}

func (e *Error) Error() string { return e.err.Error() }
func (e *Error) Unwrap() error { return e.err }

// TerminalError marks err as unrecoverable without operator action.
func TerminalError(err error) error {
	return &Error{Terminal: true, err: err}
}

// RateLimitedError marks err as rate-limited, to retry after retryAfter.
func RateLimitedError(err error, retryAfter time.Duration) error {
	return &Error{RetryAfter: retryAfter, err: err}
}
