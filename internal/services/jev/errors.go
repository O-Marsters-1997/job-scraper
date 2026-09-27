package jev

import "time"

type FailureKind int

const (
	FailureRetryable FailureKind = iota
	FailureTerminal
	FailureRateLimited
)

// Error classifies a Jev call failure for the caller's retry scheduling.
type Error struct {
	Kind       FailureKind
	RetryAfter time.Duration
	err        error
}

func (e *Error) Error() string { return e.err.Error() }
func (e *Error) Unwrap() error { return e.err }

// TerminalError marks err as unrecoverable without operator action.
func TerminalError(err error) error {
	return &Error{Kind: FailureTerminal, err: err}
}

// RateLimitedError marks err as rate-limited, to retry after retryAfter.
func RateLimitedError(err error, retryAfter time.Duration) error {
	return &Error{Kind: FailureRateLimited, RetryAfter: retryAfter, err: err}
}
