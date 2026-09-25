package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type unusedApplicationProvider struct{ providers.ApplicationProvider }

func (unusedApplicationProvider) CreateApplication(context.Context, dto.CreateApplicationInput) (dto.Application, error) {
	panic("invalid date reached provider")
}
func (unusedApplicationProvider) UpdateApplication(context.Context, dto.UpdateApplicationInput) (dto.Application, error) {
	panic("invalid date reached provider")
}

func TestApplicationHandlersRejectInvalidDate(t *testing.T) {
	h := NewApplicationHandler(unusedApplicationProvider{})
	for _, tc := range []struct {
		name   string
		handle http.HandlerFunc
		body   string
	}{
		{"create", h.CreateApplication, `{"job_id":"job-1","applied_at":"bad"}`},
		{"update", h.UpdateApplication, `{"applied_at":"bad"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/applications", strings.NewReader(tc.body))
			req = req.WithContext(auth.WithSession(req.Context(), dto.Session{UserID: "user-1"}))
			w := httptest.NewRecorder()
			tc.handle(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}
