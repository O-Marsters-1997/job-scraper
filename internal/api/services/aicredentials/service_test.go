package aicredentials_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type fakeStore struct {
	saved       map[string]string
	deletedKeys []string
}

func newFakeStore() *fakeStore { return &fakeStore{saved: map[string]string{}} }

func (f *fakeStore) Save(_ context.Context, userID, provider, plainKey string) error {
	f.saved[userID+":"+provider] = plainKey
	return nil
}

func (f *fakeStore) Delete(_ context.Context, userID, provider string) error {
	f.deletedKeys = append(f.deletedKeys, userID+":"+provider)
	return nil
}

func TestUpdateRequiresProvider(t *testing.T) {
	svc := aicredentials.New(newFakeStore())
	_, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestUpdateSavesAQuotedKeyTrimmed(t *testing.T) {
	store := newFakeStore()
	svc := aicredentials.New(store)
	key := `"sk-test"`
	if _, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic", APIKey: &key}); err != nil {
		t.Fatal(err)
	}
	if store.saved["user-1:anthropic"] != "sk-test" {
		t.Fatalf("saved = %+v", store.saved)
	}
}

func TestUpdateDeletesWhenKeyIsNil(t *testing.T) {
	store := newFakeStore()
	svc := aicredentials.New(store)
	if _, err := svc.Update(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic"}); err != nil {
		t.Fatal(err)
	}
	if len(store.deletedKeys) != 1 || store.deletedKeys[0] != "user-1:anthropic" {
		t.Fatalf("deletedKeys = %v", store.deletedKeys)
	}
}
