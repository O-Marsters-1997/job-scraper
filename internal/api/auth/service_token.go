package auth

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// ServiceTokenMiddleware authenticates machine-to-machine requests via a static
// bearer token read from INGEST_SERVICE_TOKEN.
func ServiceTokenMiddleware(next http.Handler) http.Handler {
	token := os.Getenv("INGEST_SERVICE_TOKEN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			writeUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
