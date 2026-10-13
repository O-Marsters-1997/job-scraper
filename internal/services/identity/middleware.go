package identity

import (
	"context"
	"net"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

type sessionGetter interface {
	GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (dto.Session, error)
}

func sessionMiddleware(sg sessionGetter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				writeUnauthorized(w)
				return
			}
			session, err := sg.GetSessionByTokenHash(r.Context(), hashSessionToken(cookie.Value))
			if err != nil {
				writeUnauthorized(w)
				return
			}
			ctx := handlers.WithSession(r.Context(), session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}

func limitByRemoteHost(l *attemptLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			if !l.Allow(host) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"too many attempts"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
