package auth

import (
	"context"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type contextKey struct{}

var ctxKeySession = contextKey{}

func SessionFromContext(ctx context.Context) (dto.Session, bool) {
	s, ok := ctx.Value(ctxKeySession).(dto.Session)
	return s, ok
}

func WithSession(ctx context.Context, s dto.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession, s)
}

// Middleware authenticates requests using the session_id cookie.
func Middleware(sp providers.SessionProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				WriteUnauthorized(w)
				return
			}
			session, err := sp.GetSession(r.Context(), cookie.Value)
			if err != nil {
				WriteUnauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeySession, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WriteUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
