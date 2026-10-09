package handlers

import (
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// RequireAdmin responds 403 unless the session attached by the auth
// middleware has the admin role. Mount it after that middleware.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := Session(r)
		if !ok {
			writeError(w, r, apperr.Unauthorized("unauthorized"))
			return
		}
		if s.Role != dto.RoleAdmin {
			writeError(w, r, apperr.Forbidden("admin only"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
