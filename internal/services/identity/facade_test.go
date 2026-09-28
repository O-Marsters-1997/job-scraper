package identity_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func setEncryptionKey(t *testing.T) {
	t.Helper()
	t.Setenv("AI_CREDENTIAL_ENC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
}

func buildModule(t *testing.T, st *identitytest.FakeStore, gc *fakeGoogleClient) *identity.Module {
	t.Helper()
	setEncryptionKey(t)
	m, err := identity.Build(identity.Deps{
		Store:         st,
		Seeder:        &seeder{},
		AICredentials: st,
		Profile:       st,
		GoogleClient:  gc,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestGetReturnsTheUsersDecryptedCredential(t *testing.T) {
	st := identitytest.NewFakeStore()
	m := buildModule(t, st, newFakeGoogleClient())

	creds, err := aicredentials.New(st)
	if err != nil {
		t.Fatal(err)
	}
	key := `"sk-test"`
	if _, err := creds.Update(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic", APIKey: &key}); err != nil {
		t.Fatal(err)
	}

	got, err := m.Get(context.Background(), "user-1", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-test" {
		t.Fatalf("Get(...) = %q, want sk-test", got)
	}
}

func TestGetProfileReturnsTheStoredProfile(t *testing.T) {
	st := identitytest.NewFakeStore()
	m := buildModule(t, st, newFakeGoogleClient())

	user, err := st.CreateUser(context.Background(), "bob", "hash", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}

	got, err := m.GetProfile(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "bob" || got.Email != "bob@example.com" {
		t.Fatalf("GetProfile(...) = %+v", got)
	}
}

func TestDocsClientExposesTheGoogleDocsSurface(t *testing.T) {
	gc := newFakeGoogleClient()
	gc.tabs = []google.Tab{{ID: "t.0", Title: "Resume"}}
	m := buildModule(t, identitytest.NewFakeStore(), gc)

	tabs, err := m.DocsClient().ListTabs(context.Background(), "user-1", "doc-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tabs) != 1 || tabs[0].Title != "Resume" {
		t.Fatalf("ListTabs(...) = %+v", tabs)
	}
}
