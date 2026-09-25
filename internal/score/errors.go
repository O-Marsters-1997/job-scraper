package score

import "time"

type FailureKind int

const (
	FailureRetryable FailureKind = iota
	FailureTerminal
	FailureRateLimited
)

// ScorerError classifies a scoring failure so the outbox can decide how to
// reschedule it, instead of applying the same backoff to every error.
type ScorerError struct {
	Kind       FailureKind
	RetryAfter time.Duration
	err        error
}

func (e *ScorerError) Error() string { return e.err.Error() }
func (e *ScorerError) Unwrap() error { return e.err }

// TerminalScoreError marks err as unrecoverable without operator action
// (e.g. a bad key or exhausted credit): the outbox fails the effect after
// this one attempt instead of retrying.
func TerminalScoreError(err error) error {
	return &ScorerError{Kind: FailureTerminal, err: err}
}

// RateLimitedScoreError marks err as a rate limit: the outbox schedules the
// retry at retryAfter instead of the exponential backoff.
func RateLimitedScoreError(err error, retryAfter time.Duration) error {
	return &ScorerError{Kind: FailureRateLimited, RetryAfter: retryAfter, err: err}
}
