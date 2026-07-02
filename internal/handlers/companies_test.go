package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestCompaniesHandler_List(t *testing.T) {
	companies := providers.NewMockCompanyProvider()
	_, _ = companies.UpsertCompany(t.Context(), "acme", "Acme", "greenhouse", "acme")
	targets := providers.NewMockSourceTargetProvider()
	h := NewCompaniesHandler(companies, targets, nil)

	req := httptest.NewRequest(http.MethodGet, "/companies", nil)
	req = withSession(req, "user-1")
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var got []dto.Company
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("want 1 company, got %d", len(got))
	}
}

func TestCompaniesHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "resolves and tracks a valid board URL",
			body:       map[string]any{"url": "https://boards.greenhouse.io/acmecorp"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "rejects an unresolvable URL",
			body:       map[string]any{"url": "https://example.com/careers"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "rejects missing url",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			companies := providers.NewMockCompanyProvider()
			targets := providers.NewMockSourceTargetProvider()
			h := NewCompaniesHandler(companies, targets, nil)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("want %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestCompaniesHandler_SetTracking(t *testing.T) {
	t.Run("enables tracking for an ATS company", func(t *testing.T) {
		companies := providers.NewMockCompanyProvider()
		company, _ := companies.UpsertCompany(t.Context(), "acme", "Acme", "greenhouse", "acme")
		targets := providers.NewMockSourceTargetProvider()
		h := NewCompaniesHandler(companies, targets, nil)

		body, _ := json.Marshal(map[string]bool{"enabled": true})
		req := httptest.NewRequest(http.MethodPut, "/companies/"+company.ID+"/tracking", bytes.NewReader(body))
		req = withSession(req, "user-1")
		req = withRouteID(req, company.ID)
		w := httptest.NewRecorder()

		h.SetTracking(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects tracking a discovery-only company", func(t *testing.T) {
		companies := providers.NewMockCompanyProvider()
		company, _ := companies.UpsertCompany(t.Context(), "acme", "Acme", "", "")
		targets := providers.NewMockSourceTargetProvider()
		h := NewCompaniesHandler(companies, targets, nil)

		body, _ := json.Marshal(map[string]bool{"enabled": true})
		req := httptest.NewRequest(http.MethodPut, "/companies/"+company.ID+"/tracking", bytes.NewReader(body))
		req = withSession(req, "user-1")
		req = withRouteID(req, company.ID)
		w := httptest.NewRecorder()

		h.SetTracking(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("want 422, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 for unknown company", func(t *testing.T) {
		companies := providers.NewMockCompanyProvider()
		targets := providers.NewMockSourceTargetProvider()
		h := NewCompaniesHandler(companies, targets, nil)

		body, _ := json.Marshal(map[string]bool{"enabled": true})
		req := httptest.NewRequest(http.MethodPut, "/companies/missing/tracking", bytes.NewReader(body))
		req = withSession(req, "user-1")
		req = withRouteID(req, "missing")
		w := httptest.NewRecorder()

		h.SetTracking(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("want 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}
