package aicredentials_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/aicredentials"
)

var errNotFound = errors.New("not found")

type fakeStore struct {
	saved       map[string]string
	deletedKeys []string
}

func newFakeStore() *fakeStore { return &fakeStore{saved: map[string]string{}} }

func (f *fakeStore) UpsertUserAICredential(_ context.Context, userID, provider, encKey string) error {
	f.saved[userID+"/"+provider] = encKey
	return nil
}

func (f *fakeStore) GetUserAICredential(_ context.Context, userID, provider string) (string, error) {
	v, ok := f.saved[userID+"/"+provider]
	if !ok {
		return "", errNotFound
	}
	return v, nil
}

func (f *fakeStore) DeleteUserAICredential(_ context.Context, userID, provider string) error {
	f.deletedKeys = append(f.deletedKeys, userID+"/"+provider)
	delete(f.saved, userID+"/"+provider)
	return nil
}

func (f *fakeStore) ListUserAICredentialProviders(_ context.Context, userID string) ([]string, error) {
	var out []string
	prefix := userID + "/"
	for k := range f.saved {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, k[len(prefix):])
		}
	}
	return out, nil
}

func testService(t *testing.T, store aicredentials.Store) *aicredentials.Service {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("AI_CREDENTIAL_ENC_KEY", key)
	svc, err := aicredentials.New(store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc
}

func TestNew_MissingKey(t *testing.T) {
	t.Setenv("AI_CREDENTIAL_ENC_KEY", "")
	if _, err := aicredentials.New(newFakeStore()); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestNew_WrongLength(t *testing.T) {
	t.Setenv("AI_CREDENTIAL_ENC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 16)))
	if _, err := aicredentials.New(newFakeStore()); err == nil {
		t.Fatal("expected error for wrong key length")
	}
}

func TestUpdateRequiresProvider(t *testing.T) {
	svc := testService(t, newFakeStore())
	_, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestUpdateSavesAQuotedKeyTrimmedAndGetDecrypts(t *testing.T) {
	svc := testService(t, newFakeStore())
	key := `"sk-test"`
	if _, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic", APIKey: &key}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), "user-1", "anthropic")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sk-test" {
		t.Fatalf("got %q, want sk-test", got)
	}
}

func TestUpdateDeletesWhenKeyIsNil(t *testing.T) {
	store := newFakeStore()
	svc := testService(t, store)
	if _, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic"}); err != nil {
		t.Fatal(err)
	}
	if len(store.deletedKeys) != 1 || store.deletedKeys[0] != "user-1/anthropic" {
		t.Fatalf("deletedKeys = %v", store.deletedKeys)
	}
}

func TestGetNotFound(t *testing.T) {
	svc := testService(t, newFakeStore())
	if _, err := svc.Get(context.Background(), "user-1", "anthropic"); !errors.Is(err, errNotFound) {
		t.Fatalf("err = %v, want errNotFound", err)
	}
}

func TestListProviders(t *testing.T) {
	svc := testService(t, newFakeStore())
	key := `"sk-test"`
	if _, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "openrouter", APIKey: &key}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.ListProviders(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(got) != 1 || got[0] != "openrouter" {
		t.Fatalf("got = %v, want [openrouter]", got)
	}
}
