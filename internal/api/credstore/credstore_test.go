package credstore_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/credstore"
)

type mapStore struct {
	m map[string]string
}

func newMapStore() *mapStore { return &mapStore{m: make(map[string]string)} }

func (s *mapStore) UpsertUserAICredential(_ context.Context, userID, provider, encKey string) error {
	s.m[userID+"/"+provider] = encKey
	return nil
}
func (s *mapStore) GetUserAICredential(_ context.Context, userID, provider string) (string, error) {
	v, ok := s.m[userID+"/"+provider]
	if !ok {
		return "", credstore.ErrNotFound
	}
	return v, nil
}
func (s *mapStore) DeleteUserAICredential(_ context.Context, userID, provider string) error {
	delete(s.m, userID+"/"+provider)
	return nil
}
func (s *mapStore) ListUserAICredentialProviders(_ context.Context, userID string) ([]string, error) {
	var out []string
	prefix := userID + "/"
	for k := range s.m {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, k[len(prefix):])
		}
	}
	return out, nil
}

func testStore(t *testing.T) *credstore.EnvCredentialStore {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("AI_CREDENTIAL_ENC_KEY", key)
	cs, err := credstore.New(newMapStore())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return cs
}

func TestRoundTrip(t *testing.T) {
	cs := testStore(t)
	ctx := context.Background()

	if err := cs.Save(ctx, "user1", "anthropic", "sk-secret"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := cs.Get(ctx, "user1", "anthropic")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sk-secret" {
		t.Fatalf("got %q, want %q", got, "sk-secret")
	}

	if err := cs.Delete(ctx, "user1", "anthropic"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := cs.Get(ctx, "user1", "anthropic"); err != credstore.ErrNotFound {
		t.Fatalf("after Delete want ErrNotFound, got %v", err)
	}
}

func TestNew_MissingKey(t *testing.T) {
	t.Setenv("AI_CREDENTIAL_ENC_KEY", "")
	if _, err := credstore.New(newMapStore()); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestNew_WrongLength(t *testing.T) {
	t.Setenv("AI_CREDENTIAL_ENC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 16)))
	if _, err := credstore.New(newMapStore()); err == nil {
		t.Fatal("expected error for wrong key length")
	}
}
