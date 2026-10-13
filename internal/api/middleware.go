package api

import (
	"encoding/json"
	"mime"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

const maxBodyBytes = 1 << 20

func limitBody(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > max {
				writeAppError(w, apperr.TooLarge("request body too large"))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}

func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			if r.ContentLength != 0 {
				mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || mt != "application/json" {
					writeAppError(w, apperr.UnsupportedMedia("content type must be application/json"))
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeAppError(w http.ResponseWriter, err error) {
	status, _ := apperr.StatusFor(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}
