package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

// mockCredStore is a minimal in-memory CredentialStore for handler tests.
// It stores plaintext so tests can assert the key was stored without going
// through real AES-GCM encryption.
type mockCredStore struct {
	mu   sync.Mutex
	data map[string]string // key: userID+"/"+provider

	SaveErr   error
	DeleteErr error
	ListErr   error
}

func newMockCredStore() *mockCredStore {
	return &mockCredStore{data: make(map[string]string)}
}

func (m *mockCredStore) Save(_ context.Context, userID, provider, plainKey string) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[userID+"/"+provider] = plainKey
	return nil
}

func (m *mockCredStore) Get(_ context.Context, userID, provider string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[userID+"/"+provider], nil
}

func (m *mockCredStore) Delete(_ context.Context, userID, provider string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, userID+"/"+provider)
	return nil
}

func (m *mockCredStore) ListProviders(_ context.Context, userID string) ([]string, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := userID + "/"
	var out []string
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k[len(prefix):])
		}
	}
	return out, nil
}

func TestAICredentialsHandler_UpsertCredential(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*mockCredStore)
		body       string
		wantStatus int
		check      func(*testing.T, *mockCredStore, *httptest.ResponseRecorder)
	}{
		{
			name:       "save stores credential",
			body:       `{"provider":"anthropic","apiKey":"sk-ant-secret"}`,
			wantStatus: http.StatusNoContent,
			check: func(t *testing.T, store *mockCredStore, w *httptest.ResponseRecorder) {
				store.mu.Lock()
				stored := store.data["user-1/anthropic"]
				store.mu.Unlock()
				if stored == "" {
					t.Error("expected credential to be stored")
				}
				if strings.Contains(w.Body.String(), "sk-ant-secret") {
					t.Error("response must not echo the API key")
				}
			},
		},
		{
			name: "delete removes credential",
			setup: func(store *mockCredStore) {
				store.data["user-1/anthropic"] = "sk-ant-old"
			},
			body:       `{"provider":"anthropic","apiKey":null}`,
			wantStatus: http.StatusNoContent,
			check: func(t *testing.T, store *mockCredStore, _ *httptest.ResponseRecorder) {
				store.mu.Lock()
				_, exists := store.data["user-1/anthropic"]
				store.mu.Unlock()
				if exists {
					t.Error("credential should have been deleted")
				}
			},
		},
		{
			name:       "invalid json returns 400",
			body:       "not-json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing provider returns 400",
			body:       `{"apiKey":"sk-ant-foo"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMockCredStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			h := NewAICredentialsHandler(store)

			req := httptest.NewRequest(http.MethodPut, "/ai-credentials", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.UpsertCredential(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.check != nil {
				tt.check(t, store, w)
			}
		})
	}
}

func TestAIPrefsHandler_GetConfiguredProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*mockCredStore)
		wantStatus int
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "lists configured providers",
			setup: func(s *mockCredStore) {
				s.data["user-1/anthropic"] = "encrypted"
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var got aiPrefsResponse
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(got.ConfiguredProviders) != 1 || got.ConfiguredProviders[0] != "anthropic" {
					t.Errorf("configuredProviders = %v; want [anthropic]", got.ConfiguredProviders)
				}
			},
		},
		{
			name: "does not leak key material",
			setup: func(s *mockCredStore) {
				s.data["user-1/anthropic"] = "sk-ant-topsecret"
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				if strings.Contains(w.Body.String(), "sk-ant-topsecret") {
					t.Error("response body must not contain API key material")
				}
			},
		},
		{
			name:       "returns empty slice when no credentials",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var got aiPrefsResponse
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if got.ConfiguredProviders == nil {
					t.Error("configuredProviders should be an empty slice, not null")
				}
				if len(got.ConfiguredProviders) != 0 {
					t.Errorf("configuredProviders = %v; want []", got.ConfiguredProviders)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMockCredStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			prefsStore := providers.NewMockUserAIPrefsProvider()
			h := NewAIPrefsHandler(prefsStore, store)

			req := httptest.NewRequest(http.MethodGet, "/ai-prefs", nil)
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()
			h.GetAIPrefs(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.check != nil {
				tt.check(t, w)
			}
		})
	}
}
