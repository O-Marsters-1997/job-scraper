package identity_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
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

func TestUserIDByUsername(t *testing.T) {
	st := identitytest.NewFakeStore()
	m := identity.Build(testDeps(t, identity.Deps{Store: st}))
	user, err := st.CreateUser(t.Context(), "bob", "hash", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}

	got, err := m.UserIDByUsername(t.Context(), "bob")
	if err != nil || got != user.ID {
		t.Fatalf("UserIDByUsername(bob) = %q, %v, want %q", got, err, user.ID)
	}
	if _, err := m.UserIDByUsername(t.Context(), "nobody"); !errors.Is(err, data.ErrNotFound) {
		t.Errorf("UserIDByUsername(nobody) err = %v, want ErrNotFound", err)
	}
}

func TestSetRole(t *testing.T) {
	st := identitytest.NewFakeStore()
	m := identity.Build(testDeps(t, identity.Deps{Store: st}))
	if _, err := st.CreateUser(t.Context(), "bob", "hash", ""); err != nil {
		t.Fatal(err)
	}

	if err := m.SetRole(t.Context(), "bob", "admin"); err != nil {
		t.Errorf("SetRole(bob, admin) err = %v, want nil", err)
	}
	err := m.SetRole(t.Context(), "bob", "root")
	if status, _ := apperr.StatusFor(err); status != http.StatusBadRequest {
		t.Errorf("SetRole(bob, root) status = %d (err %v), want 400", status, err)
	}
	if err = m.SetRole(t.Context(), "nobody", "admin"); !errors.Is(err, data.ErrNotFound) {
		t.Errorf("SetRole(nobody, admin) err = %v, want ErrNotFound", err)
	}
}

func TestResetPassword(t *testing.T) {
	st := identitytest.NewFakeStore()
	deps := testDeps(t, identity.Deps{Store: st})
	m, svc := identity.Build(deps), identity.NewService(deps)
	if _, _, err := svc.Signup(t.Context(), "bob", "old-password", ""); err != nil {
		t.Fatal(err)
	}

	t.Run("new password logs in and old one does not", func(t *testing.T) {
		if err := m.ResetPassword(t.Context(), "bob", "new-password"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := svc.Login(t.Context(), "bob", "new-password"); err != nil {
			t.Errorf("Login(bob, new-password) err = %v, want nil", err)
		}
		_, _, err := svc.Login(t.Context(), "bob", "old-password")
		if status, _ := apperr.StatusFor(err); status != http.StatusUnauthorized {
			t.Errorf("Login(bob, old-password) status = %d (err %v), want 401", status, err)
		}
	})

	t.Run("empty password is invalid", func(t *testing.T) {
		err := m.ResetPassword(t.Context(), "bob", "")
		if status, _ := apperr.StatusFor(err); status != http.StatusBadRequest {
			t.Errorf("ResetPassword(bob, \"\") status = %d (err %v), want 400", status, err)
		}
	})

	t.Run("unknown user is not found", func(t *testing.T) {
		if err := m.ResetPassword(t.Context(), "nobody", "pw"); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("ResetPassword(nobody) err = %v, want ErrNotFound", err)
		}
	})
}
