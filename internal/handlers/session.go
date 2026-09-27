package handlers

import (
	"context"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type contextKey struct{}

var sessionKey contextKey

// WithSession attaches a session to ctx for downstream handlers to read via
// Session or UserID.
func WithSession(ctx context.Context, s dto.Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// Session returns the session attached to r by the auth middleware.
func Session(r *http.Request) (dto.Session, bool) {
	s, ok := r.Context().Value(sessionKey).(dto.Session)
	return s, ok
}

// UserID returns the authenticated caller's user ID, or an unauthorized
// error if no session is attached to r.
func UserID(r *http.Request) (string, error) {
	s, ok := Session(r)
	if !ok {
		return "", apperr.Unauthorized("unauthorized")
	}
	return s.UserID, nil
}
