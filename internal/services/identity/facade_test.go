package identity_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func TestGetReturnsTheUsersDecryptedCredential(t *testing.T) {
	st := identitytest.NewFakeStore()
	m := identity.Build(testDeps(t, identity.Deps{Store: st}))

	saveKey(t, identity.NewService(testDeps(t, identity.Deps{Store: st})), "anthropic", `"sk-test"`)

	got, err := m.Get(t.Context(), "user-1", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-test" {
		t.Fatalf("Get(...) = %q, want sk-test", got)
	}
}

func TestGetProfileReturnsTheStoredProfile(t *testing.T) {
	st := identitytest.NewFakeStore()
	m := identity.Build(testDeps(t, identity.Deps{Store: st}))

	user, err := st.CreateUser(t.Context(), "bob", "hash", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}

	got, err := m.GetProfile(t.Context(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "bob" || got.Email != "bob@example.com" {
		t.Fatalf("GetProfile(...) = %+v", got)
	}
}
