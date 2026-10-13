package identity_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
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

	t.Run("old session no longer authenticates", func(t *testing.T) {
		_, user, err := svc.Signup(t.Context(), "carl", "old-password", "")
		if err != nil {
			t.Fatal(err)
		}
		tokenHash := sha256.Sum256([]byte("carl"))
		if _, err := st.CreateSession(t.Context(), user.ID, tokenHash[:], time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := m.ResetPassword(t.Context(), "carl", "new-password"); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetSessionByTokenHash(t.Context(), tokenHash[:]); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("GetSessionByTokenHash(carl) err = %v, want ErrNotFound", err)
		}
	})

	for _, pw := range []string{"", "1234567", strings.Repeat("a", 73)} {
		t.Run(fmt.Sprintf("password of %d bytes is invalid", len(pw)), func(t *testing.T) {
			err := m.ResetPassword(t.Context(), "bob", pw)
			if status, _ := apperr.StatusFor(err); status != http.StatusBadRequest {
				t.Errorf("ResetPassword(bob, %d bytes) status = %d (err %v), want 400", len(pw), status, err)
			}
		})
	}

	t.Run("repeated resets for one user are rate limited", func(t *testing.T) {
		now := time.Unix(1_700_000_000, 0)
		m := identity.Build(testDeps(t, identity.Deps{Now: func() time.Time { return now }}))
		var err error
		for range 6 {
			err = m.ResetPassword(t.Context(), "nobody", "password1")
		}
		if !apperr.IsKind(err, apperr.KindTooManyRequests) {
			t.Errorf("6th ResetPassword err = %v, want KindTooManyRequests", err)
		}
	})

	t.Run("admin user creation rejects short passwords and accepts valid ones", func(t *testing.T) {
		m := identity.Build(testDeps(t, identity.Deps{}))
		if _, err := m.CreateUser(t.Context(), dto.CreateUserInput{Username: "cat", Password: "short"}); !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("CreateUser(short) err = %v, want KindInvalid", err)
		}
		if _, err := m.CreateUser(t.Context(), dto.CreateUserInput{Username: "cat", Password: "long-enough"}); err != nil {
			t.Fatalf("CreateUser(long-enough) err = %v", err)
		}
	})

	t.Run("unknown user is not found", func(t *testing.T) {
		if err := m.ResetPassword(t.Context(), "nobody", "password1"); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("ResetPassword(nobody) err = %v, want ErrNotFound", err)
		}
	})
}
