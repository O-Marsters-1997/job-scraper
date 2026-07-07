package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestScoringConfigHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupStore func(*providers.MockSearchConfigProvider)
		userID     string
		wantStatus int
		wantResp   *scoringConfigResponse
	}{
		{
			name:       "no config returns zero-value defaults",
			userID:     "user-1",
			wantStatus: http.StatusOK,
			wantResp:   emptyScoringConfigResponse(),
		},
		{
			name: "provider error returns 500",
			setupStore: func(s *providers.MockSearchConfigProvider) {
				s.GetErr = errors.New("db down")
			},
			userID:     "user-1",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "user sees only their own config",
			setupStore: func(s *providers.MockSearchConfigProvider) {
				_, _ = s.UpsertSearchConfig(context.Background(), dto.SearchConfig{
					UserID:            "user-a",
					SuitabilityRubric: "user-a rubric",
					NotifyThreshold:   80,
				})
			},
			userID:     "user-b",
			wantStatus: http.StatusOK,
			wantResp:   emptyScoringConfigResponse(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockSearchConfigProvider()
			if tt.setupStore != nil {
				tt.setupStore(store)
			}
			h := NewScoringConfigHandler(store)

			req := httptest.NewRequest(http.MethodGet, "/scoring-config", nil)
			req = withSession(req, tt.userID)
			w := httptest.NewRecorder()

			h.GetScoringConfig(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantResp != nil {
				var got scoringConfigResponse
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if !reflect.DeepEqual(got, *tt.wantResp) {
					t.Errorf("got %+v; want %+v", got, *tt.wantResp)
				}
			}
		})
	}
}

// emptyScoringConfigResponse is what a zero-value dto.SearchConfig encodes
// as: exclusion lists are empty arrays, never null.
func emptyScoringConfigResponse() *scoringConfigResponse {
	return &scoringConfigResponse{
		ExcludedTitleKeywords: []string{},
		ExcludedCompanies:     []string{},
		ExcludedSeniority:     []string{},
		ExcludedLocations:     []string{},
	}
}

func TestScoringConfigHandler_Put(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "bad body returns 400",
			body:       "not-json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid body returns 200",
			body:       `{"suitabilityRubric":"Looking for senior Go engineers","notifyThreshold":75}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown seniority level returns 400",
			body:       `{"excludedSeniority":["ceo"]}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockSearchConfigProvider()
			h := NewScoringConfigHandler(store)

			req := httptest.NewRequest(http.MethodPut, "/scoring-config", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.UpdateScoringConfig(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestScoringConfigHandler_PutThenGet(t *testing.T) {
	t.Parallel()

	store := providers.NewMockSearchConfigProvider()
	h := NewScoringConfigHandler(store)

	putBody, _ := json.Marshal(map[string]any{
		"suitabilityRubric":     "Looking for senior Go engineers",
		"notifyThreshold":       75,
		"excludedTitleKeywords": []string{"Java", "Sales"},
		"excludedCompanies":     []string{"Acme Corp"},
		"excludedSeniority":     []string{"intern", "junior"},
		"excludedLocations":     []string{"United States"},
	})
	putReq := httptest.NewRequest(http.MethodPut, "/scoring-config", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq = withSession(putReq, "user-1")
	putW := httptest.NewRecorder()

	h.UpdateScoringConfig(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT: want 200, got %d: %s", putW.Code, putW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/scoring-config", nil)
	getReq = withSession(getReq, "user-1")
	getW := httptest.NewRecorder()

	h.GetScoringConfig(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET: want 200, got %d", getW.Code)
	}
	var got scoringConfigResponse
	if err := json.NewDecoder(getW.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := scoringConfigResponse{
		SuitabilityRubric:     "Looking for senior Go engineers",
		NotifyThreshold:       75,
		ExcludedTitleKeywords: []string{"java", "sales"},
		ExcludedCompanies:     []string{"acme corp"},
		ExcludedSeniority:     []string{"intern", "junior"},
		ExcludedLocations:     []string{"united states"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v; want %+v", got, want)
	}
}
