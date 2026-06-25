package handlers

import (
	"bytes"
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

func TestAICredentialsHandler_Put_Save(t *testing.T) {
	t.Parallel()

	store := newMockCredStore()
	h := NewAICredentialsHandler(store)

	body, _ := json.Marshal(map[string]any{"provider": "anthropic", "apiKey": "sk-ant-secret"})
	req := httptest.NewRequest(http.MethodPut, "/ai-credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, "user-1")
	w := httptest.NewRecorder()

	h.Put(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d; want 204: %s", w.Code, w.Body.String())
	}
	store.mu.Lock()
	stored := store.data["user-1/anthropic"]
	store.mu.Unlock()
	if stored == "" {
		t.Error("expected credential to be stored")
	}
	if strings.Contains(w.Body.String(), "sk-ant-secret") {
		t.Error("response must not echo the API key")
	}
}

func TestAICredentialsHandler_Put_Delete(t *testing.T) {
	t.Parallel()

	store := newMockCredStore()
	store.data["user-1/anthropic"] = "sk-ant-old"

	h := NewAICredentialsHandler(store)

	body, _ := json.Marshal(map[string]any{"provider": "anthropic", "apiKey": nil})
	req := httptest.NewRequest(http.MethodPut, "/ai-credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSession(req, "user-1")
	w := httptest.NewRecorder()

	h.Put(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d; want 204: %s", w.Code, w.Body.String())
	}
	store.mu.Lock()
	_, exists := store.data["user-1/anthropic"]
	store.mu.Unlock()
	if exists {
		t.Error("credential should have been deleted")
	}
}

func TestAICredentialsHandler_Put_BadRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"invalid json", "not-json"},
		{"missing provider", `{"apiKey":"sk-ant-foo"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMockCredStore()
			h := NewAICredentialsHandler(store)

			req := httptest.NewRequest(http.MethodPut, "/ai-credentials", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req = withSession(req, "user-1")
			w := httptest.NewRecorder()

			h.Put(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d; want 400", w.Code)
			}
		})
	}
}

func TestAIPrefsHandler_Get_ConfiguredProviders(t *testing.T) {
	t.Parallel()

	store := newMockCredStore()
	store.data["user-1/anthropic"] = "encrypted"

	prefsStore := providers.NewMockUserAIPrefsProvider()
	h := NewAIPrefsHandler(prefsStore, store)

	req := httptest.NewRequest(http.MethodGet, "/ai-prefs", nil)
	req = withSession(req, "user-1")
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200: %s", w.Code, w.Body.String())
	}
	var got aiPrefsResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.ConfiguredProviders) != 1 || got.ConfiguredProviders[0] != "anthropic" {
		t.Errorf("configuredProviders = %v; want [anthropic]", got.ConfiguredProviders)
	}
}

func TestAIPrefsHandler_Get_NoKeyMaterial(t *testing.T) {
	t.Parallel()

	store := newMockCredStore()
	// Store the key so it exists; value should never appear in the HTTP response.
	store.data["user-1/anthropic"] = "sk-ant-topsecret"

	prefsStore := providers.NewMockUserAIPrefsProvider()
	h := NewAIPrefsHandler(prefsStore, store)

	req := httptest.NewRequest(http.MethodGet, "/ai-prefs", nil)
	req = withSession(req, "user-1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if strings.Contains(w.Body.String(), "sk-ant-topsecret") {
		t.Error("response body must not contain API key material")
	}
}

func TestAIPrefsHandler_Get_EmptyWhenNoCredentials(t *testing.T) {
	t.Parallel()

	store := newMockCredStore() // no entries
	prefsStore := providers.NewMockUserAIPrefsProvider()
	h := NewAIPrefsHandler(prefsStore, store)

	req := httptest.NewRequest(http.MethodGet, "/ai-prefs", nil)
	req = withSession(req, "user-1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
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
}
