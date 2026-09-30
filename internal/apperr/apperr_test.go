package apperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

func TestStatusFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"invalid", apperr.Invalid("bad"), http.StatusBadRequest},
		{"unauthorized", apperr.Unauthorized("nope"), http.StatusUnauthorized},
		{"not found", apperr.NotFound("gone"), http.StatusNotFound},
		{"conflict", apperr.Conflict("dup"), http.StatusConflict},
		{"unprocessable", apperr.Unprocessable("nope"), http.StatusUnprocessableEntity},
		{"upstream", apperr.Upstream("dep down"), http.StatusBadGateway},
		{"unavailable", apperr.Unavailable("retry"), http.StatusServiceUnavailable},
		{"wrapped", fmt.Errorf("service.Create: %w", apperr.Conflict("dup")), http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, ok := apperr.StatusFor(tc.err)
			if !ok {
				t.Fatalf("StatusFor(%v) ok = false, want true", tc.err)
			}
			if status != tc.want {
				t.Errorf("StatusFor(%v) = %d, want %d", tc.err, status, tc.want)
			}
		})
	}

	t.Run("plain error has no status", func(t *testing.T) {
		if status, ok := apperr.StatusFor(errors.New("plain")); ok {
			t.Errorf("StatusFor(plain error) = %d, true, want ok = false", status)
		}
	})
}

func TestErrorsIs(t *testing.T) {
	errConflict := apperr.Conflict("application already exists")
	wrapped := fmt.Errorf("db.Create: %w", errConflict)
	if !errors.Is(wrapped, errConflict) {
		t.Errorf("errors.Is(%v, %v) = false, want true", wrapped, errConflict)
	}
}
