package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
			wantResp:   &scoringConfigResponse{},
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
					RelevanceCutoff:   50,
					NotifyThreshold:   80,
				})
			},
			userID:     "user-b",
			wantStatus: http.StatusOK,
			wantResp:   &scoringConfigResponse{},
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
				if got != *tt.wantResp {
					t.Errorf("got %+v; want %+v", got, *tt.wantResp)
				}
			}
		})
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
			body:       `{"suitabilityRubric":"Looking for senior Go engineers","relevanceCutoff":30,"notifyThreshold":75}`,
			wantStatus: http.StatusOK,
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
		"suitabilityRubric": "Looking for senior Go engineers",
		"relevanceCutoff":   30,
		"notifyThreshold":   75,
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
		SuitabilityRubric: "Looking for senior Go engineers",
		RelevanceCutoff:   30,
		NotifyThreshold:   75,
	}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}
