package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func TestRequireAdmin(t *testing.T) {
	h := handlers.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name    string
		session *dto.Session
		want    int
	}{
		{"no session", nil, http.StatusUnauthorized},
		{"user", &dto.Session{UserID: "u", Role: dto.RoleUser}, http.StatusForbidden},
		{"admin", &dto.Session{UserID: "u", Role: dto.RoleAdmin}, http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.session != nil {
				req = req.WithContext(handlers.WithSession(req.Context(), *tt.session))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("RequireAdmin status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
