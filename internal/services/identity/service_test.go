package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func testDeps(t *testing.T, d identity.Deps) identity.Deps {
	t.Helper()
	if d.Store == nil {
		d.Store = identitytest.NewFakeStore()
	}
	if d.Seeder == nil {
		d.Seeder = applications.Build(applications.Deps{Store: applicationstest.NewFakeStore()})
	}
	if d.GoogleClient == nil {
		d.GoogleClient = identitytest.Unlinked(false)
	}
	if d.Cipher == nil {
		d.Cipher = identitytest.NewCipher(t)
	}
	return d
}

type failingSeeder struct{ err error }

func (s failingSeeder) SeedDefaults(context.Context, pgx.Tx, string) error { return s.err }

type failingList struct {
	identity.Store
	err error
}

func (f failingList) ListUserAICredentialProviders(context.Context, string) ([]string, error) {
	return nil, f.err
}

func seedUser(t *testing.T, st identity.Store, username, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(t.Context(), username, string(hash), ""); err != nil {
		t.Fatal(err)
	}
}

func saveKey(t *testing.T, svc *identity.Service, provider, key string) {
	t.Helper()
	if _, err := svc.UpdateCredential(t.Context(), "user-1", dto.UpsertCredentialInput{Provider: provider, APIKey: &key}); err != nil {
		t.Fatal(err)
	}
}

func TestLogin(t *testing.T) {
	ctx := t.Context()

	t.Run("valid credentials start a session", func(t *testing.T) {
		st := identitytest.NewFakeStore()
		seedUser(t, st, "alice", "secret")

		session, user, err := identity.NewService(testDeps(t, identity.Deps{Store: st})).Login(ctx, "alice", "secret")
		if err != nil {
			t.Fatalf("Login(alice) err = %v", err)
		}
		if session.UserID != user.ID || user.Username != "alice" {
			t.Errorf("Login(alice) = session %+v, user %+v", session, user)
		}
	})

	errCases := []struct{ name, username, password string }{
		{"wrong password", "alice", "wrong"},
		{"unknown user", "nobody", "secret"},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			st := identitytest.NewFakeStore()
			seedUser(t, st, "alice", "secret")

			_, _, err := identity.NewService(testDeps(t, identity.Deps{Store: st})).Login(ctx, tt.username, tt.password)
			if !apperr.IsKind(err, apperr.KindUnauthorized) {
				t.Errorf("Login(%s) err = %v, want KindUnauthorized", tt.username, err)
			}
		})
	}
}

func TestSignup(t *testing.T) {
	ctx := t.Context()

	t.Run("creates the user and seeds their default statuses", func(t *testing.T) {
		statuses := applicationstest.NewFakeStore()
		deps := testDeps(t, identity.Deps{Seeder: applications.Build(applications.Deps{Store: statuses})})

		session, user, err := identity.NewService(deps).Signup(ctx, "bob", "hunter2", "bob@example.com")
		if err != nil {
			t.Fatalf("Signup(bob) err = %v", err)
		}
		if user.Username != "bob" || session.UserID != user.ID {
			t.Errorf("Signup(bob) = session %+v, user %+v", session, user)
		}

		got, err := statuses.ListApplicationStatusesByUser(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, s := range got {
			names = append(names, s.Name)
		}
		wantNames := []string{"Saved", "Applied", "Interviewing", "Offer", "Rejected"}
		if diff := cmp.Diff(wantNames, names, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
			t.Errorf("seeded statuses mismatch (-want +got):\n%s", diff)
		}
	})

	seedErr := errors.New("seed boom")
	errCases := []struct {
		name             string
		username, passwd string
		seeder           identity.StatusSeeder
		check            func(error) bool
	}{
		{"missing credentials", "", "", nil, func(err error) bool { return apperr.IsKind(err, apperr.KindInvalid) }},
		{"duplicate username", "alice", "pass", nil, func(err error) bool { return apperr.IsKind(err, apperr.KindConflict) }},
		{"seeding fails", "bob", "hunter2", failingSeeder{seedErr}, func(err error) bool { return errors.Is(err, seedErr) }},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			st := identitytest.NewFakeStore()
			seedUser(t, st, "alice", "secret")

			session, _, err := identity.NewService(testDeps(t, identity.Deps{Store: st, Seeder: tt.seeder})).Signup(ctx, tt.username, tt.passwd, "")
			if !tt.check(err) {
				t.Errorf("Signup(%q) err = %v", tt.username, err)
			}
			if session != (dto.Session{}) {
				t.Errorf("Signup(%q) session = %+v, want none", tt.username, session)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	ctx := t.Context()
	st := identitytest.NewFakeStore()
	user, err := st.CreateUser(ctx, "dana", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if err := identity.NewService(testDeps(t, identity.Deps{Store: st})).Logout(ctx, session.ID); err != nil {
		t.Fatalf("Logout err = %v", err)
	}
	if _, err := st.GetSession(ctx, session.ID); err == nil {
		t.Error("GetSession after logout: want session gone")
	}
}

func TestUpdateCredential(t *testing.T) {
	ctx := t.Context()

	t.Run("requires a provider", func(t *testing.T) {
		_, err := identity.NewService(testDeps(t, identity.Deps{})).UpdateCredential(ctx, "user-1", dto.UpsertCredentialInput{})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("err = %v, want KindInvalid", err)
		}
	})

	t.Run("trims quotes and GetCredential decrypts", func(t *testing.T) {
		svc := identity.NewService(testDeps(t, identity.Deps{}))
		saveKey(t, svc, "anthropic", `"sk-test"`)

		got, err := svc.GetCredential(ctx, "user-1", "anthropic")
		if err != nil {
			t.Fatalf("GetCredential err = %v", err)
		}
		if got != "sk-test" {
			t.Errorf("GetCredential = %q, want sk-test", got)
		}
	})

	t.Run("nil key deletes the credential", func(t *testing.T) {
		svc := identity.NewService(testDeps(t, identity.Deps{}))
		saveKey(t, svc, "anthropic", "sk-test")
		if _, err := svc.UpdateCredential(ctx, "user-1", dto.UpsertCredentialInput{Provider: "anthropic"}); err != nil {
			t.Fatal(err)
		}

		_, err := svc.GetCredential(ctx, "user-1", "anthropic")
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("GetCredential after delete err = %v, want ErrNotFound", err)
		}
	})
}

func TestGetCredential(t *testing.T) {
	_, err := identity.NewService(testDeps(t, identity.Deps{})).GetCredential(t.Context(), "user-1", "anthropic")
	if !errors.Is(err, data.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestGetAIPrefs(t *testing.T) {
	ctx := t.Context()

	t.Run("no configured providers", func(t *testing.T) {
		got, err := identity.NewService(testDeps(t, identity.Deps{})).GetAIPrefs(ctx, "user-1")
		if err != nil {
			t.Fatalf("GetAIPrefs err = %v", err)
		}
		want := dto.AIPrefsView{ConfiguredProviders: []string{}, ScoringEnabled: false}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetAIPrefs mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("configured provider enables scoring", func(t *testing.T) {
		svc := identity.NewService(testDeps(t, identity.Deps{}))
		saveKey(t, svc, "openrouter", "sk-test")

		got, err := svc.GetAIPrefs(ctx, "user-1")
		if err != nil {
			t.Fatalf("GetAIPrefs err = %v", err)
		}
		want := dto.AIPrefsView{ConfiguredProviders: []string{"openrouter"}, ScoringEnabled: true}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetAIPrefs mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("credential list error propagates", func(t *testing.T) {
		listErr := errors.New("creds down")
		svc := identity.NewService(testDeps(t, identity.Deps{Store: failingList{Store: identitytest.NewFakeStore(), err: listErr}}))
		if _, err := svc.GetAIPrefs(ctx, "user-1"); !errors.Is(err, listErr) {
			t.Errorf("err = %v, want %v", err, listErr)
		}
	})
}

func TestUpdateProfile(t *testing.T) {
	ctx := t.Context()
	st := identitytest.NewFakeStore()
	user, err := st.CreateUser(ctx, "alice", "hash", "old@example.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := identity.NewService(testDeps(t, identity.Deps{Store: st})).UpdateProfile(ctx, user.ID, dto.UpdateProfileInput{Email: "new@example.com"}); err != nil {
		t.Fatalf("UpdateProfile err = %v", err)
	}
	got, err := st.GetProfile(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := dto.Profile{Username: "alice", Email: "new@example.com"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetProfile mismatch (-want +got):\n%s", diff)
	}
}
