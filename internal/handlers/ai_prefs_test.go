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
)

func TestAIPrefsHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupStore func(*providers.MockUserAIPrefsProvider)
		userID     string
		wantStatus int
		wantModel  string
		wantModels bool // expect non-empty availableModels
	}{
		{
			name:       "new user gets default model",
			userID:     "user-1",
			wantStatus: http.StatusOK,
			wantModel:  defaultSuitabilityModel,
			wantModels: true,
		},
		{
			name: "user sees their saved model",
			setupStore: func(s *providers.MockUserAIPrefsProvider) {
				_, _ = s.UpsertUserAIPrefs(context.Background(), "user-1", "claude-opus-4-8", "claude-sonnet-4-6")
			},
			userID:     "user-1",
			wantStatus: http.StatusOK,
			wantModel:  "claude-opus-4-8",
			wantModels: true,
		},
		{
			name: "user sees only their own prefs",
			setupStore: func(s *providers.MockUserAIPrefsProvider) {
				_, _ = s.UpsertUserAIPrefs(context.Background(), "user-a", "claude-sonnet-4-6", "claude-sonnet-4-6")
			},
			userID:     "user-b",
			wantStatus: http.StatusOK,
			wantModel:  defaultSuitabilityModel,
			wantModels: true,
		},
		{
			name: "provider error returns 500",
			setupStore: func(s *providers.MockUserAIPrefsProvider) {
				s.GetErr = errors.New("db down")
			},
			userID:     "user-1",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockUserAIPrefsProvider()
			if tt.setupStore != nil {
				tt.setupStore(store)
			}
			h := NewAIPrefsHandler(store, nil)

			req := httptest.NewRequest(http.MethodGet, "/ai-prefs", nil)
			req = withSession(req, tt.userID)
			w := httptest.NewRecorder()

			h.GetAIPrefs(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantModel == "" {
				return
			}
			var got aiPrefsResponse
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.SuitabilityModel != tt.wantModel {
				t.Errorf("suitabilityModel = %q; want %q", got.SuitabilityModel, tt.wantModel)
			}
			if tt.wantModels && len(got.AvailableModels) == 0 {
				t.Error("availableModels: want non-empty list")
			}
		})
	}
}

func TestAIPrefsHandler_Put(t *testing.T) {
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
			name:       "unknown model returns 400",
			body:       `{"suitabilityModel":"gpt-4o"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid model returns 200",
			body:       `{"suitabilityModel":"claude-sonnet-4-6"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockUserAIPrefsProvider()
			h := NewAIPrefsHandler(store, nil)

			req := httptest.NewRequest(http.MethodPut, "/ai-prefs", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.UpdateAIPrefs(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestAIPrefsHandler_PutThenGet(t *testing.T) {
	t.Parallel()

	store := providers.NewMockUserAIPrefsProvider()
	h := NewAIPrefsHandler(store, nil)

	putBody, _ := json.Marshal(map[string]string{"suitabilityModel": "claude-sonnet-4-6"})
	putReq := httptest.NewRequest(http.MethodPut, "/ai-prefs", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq = withSession(putReq, "user-1")
	putW := httptest.NewRecorder()

	h.UpdateAIPrefs(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT: want 200, got %d: %s", putW.Code, putW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/ai-prefs", nil)
	getReq = withSession(getReq, "user-1")
	getW := httptest.NewRecorder()

	h.GetAIPrefs(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET: want 200, got %d", getW.Code)
	}
	var got aiPrefsResponse
	if err := json.NewDecoder(getW.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := aiPrefsResponse{
		SuitabilityModel: "claude-sonnet-4-6",
		AvailableModels:  availableModels,
	}
	if got.SuitabilityModel != want.SuitabilityModel {
		t.Errorf("suitabilityModel = %q; want %q", got.SuitabilityModel, want.SuitabilityModel)
	}
}
