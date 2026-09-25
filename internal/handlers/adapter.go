package handlers

import (
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"reflect"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
)

func userID(r *http.Request) (string, error) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		return "", apperr.Unauthorized("unauthorized")
	}
	return session.UserID, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		_, _ = w.Write([]byte("[]"))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, ok := apperr.StatusFor(err)
	msg := err.Error()
	if !ok {
		slog.Error("unhandled handler error",
			slog.String("route", r.Method+" "+r.URL.Path),
			slog.Any("err", err),
		)
		status = http.StatusInternalServerError
		msg = "internal server error"
	}
	body := map[string]any{"error": msg}
	maps.Copy(body, apperr.FieldsFor(err))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
