package score

import "time"

type FailureKind int

const (
	FailureRetryable FailureKind = iota
	FailureTerminal
	FailureRateLimited
)

// ScorerError classifies a scoring failure for outbox retry scheduling.
type ScorerError struct {
	Kind       FailureKind
	RetryAfter time.Duration
	err        error
}

func (e *ScorerError) Error() string { return e.err.Error() }
func (e *ScorerError) Unwrap() error { return e.err }

// TerminalScoreError marks err as unrecoverable without operator action.
func TerminalScoreError(err error) error {
	return &ScorerError{Kind: FailureTerminal, err: err}
}

// RateLimitedScoreError marks err as rate-limited, to retry after retryAfter.
func RateLimitedScoreError(err error, retryAfter time.Duration) error {
	return &ScorerError{Kind: FailureRateLimited, RetryAfter: retryAfter, err: err}
}
