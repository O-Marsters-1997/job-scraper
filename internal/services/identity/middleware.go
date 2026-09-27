package identity

import (
	"context"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

type sessionGetter interface {
	GetSession(ctx context.Context, id string) (dto.Session, error)
}

func sessionMiddleware(sg sessionGetter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				writeUnauthorized(w)
				return
			}
			session, err := sg.GetSession(r.Context(), cookie.Value)
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
