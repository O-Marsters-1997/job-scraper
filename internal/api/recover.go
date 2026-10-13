package api

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			slog.ErrorContext(r.Context(), "handler panic",
				slog.String(logger.KeyMethod, r.Method),
				slog.String("path", r.URL.Path),
				slog.Any("panic", rec),
				slog.String("stack", string(debug.Stack())),
			)
			handlers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}()
		next.ServeHTTP(w, r)
	})
}
