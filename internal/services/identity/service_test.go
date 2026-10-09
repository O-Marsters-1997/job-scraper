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
	"github.com/ollymarsters/job-scraper/internal/openrouter"
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

type stubKeyUsage struct {
	info   openrouter.KeyInfo
	err    error
	calls  []string
	onCall func()
}

func (s *stubKeyUsage) KeyUsage(_ context.Context, apiKey string) (openrouter.KeyInfo, error) {
	s.calls = append(s.calls, apiKey)
	if s.onCall != nil {
		s.onCall()
	}
	return s.info, s.err
}

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func TestAIUsage(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	f := func(v float64) *float64 { return &v }
	monthly := "monthly"

	newSvc := func(t *testing.T, fetch *stubKeyUsage, clock *fakeClock) *identity.Service {
		t.Helper()
		return identity.NewService(testDeps(t, identity.Deps{KeyUsage: fetch, Now: clock.Now}))
	}

	t.Run("no saved key is not configured and makes no call", func(t *testing.T) {
		fetch := &stubKeyUsage{}
		got, err := newSvc(t, fetch, &fakeClock{start}).AIUsage(t.Context(), "user-1")
		if err != nil {
			t.Fatalf("AIUsage() err = %v", err)
		}
		if got.Status != dto.QuotaNotConfigured || got.Level != dto.UsageOK || len(fetch.calls) != 0 {
			t.Errorf("AIUsage() = %+v with %d calls, want not_configured/ok and no calls", got, len(fetch.calls))
		}
	})

	t.Run("limited key reports percent, level and monthly reset", func(t *testing.T) {
		fetch := &stubKeyUsage{info: openrouter.KeyInfo{Limit: f(10), LimitRemaining: f(0.4), LimitReset: &monthly, Usage: 9.6}}
		svc := newSvc(t, fetch, &fakeClock{start})
		saveKey(t, svc, "openrouter", "sk-or-1")

		got, err := svc.AIUsage(t.Context(), "user-1")
		if err != nil {
			t.Fatalf("AIUsage() err = %v", err)
		}
		resets := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
		want := dto.QuotaView{
			Provider: "openrouter", Status: dto.QuotaOK, Unit: "USD",
			Used: f(9.6), Limit: f(10), Percent: f(96), Level: dto.UsageCritical,
			ResetsAt: &resets, FetchedAt: &start,
		}
		if diff := cmp.Diff(want, got, cmpopts.EquateApprox(0, 1e-9)); diff != "" {
			t.Errorf("AIUsage() (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"sk-or-1"}, fetch.calls); diff != "" {
			t.Errorf("fetched keys (-want +got):\n%s", diff)
		}
	})

	t.Run("key without a limit shows monthly spend only", func(t *testing.T) {
		fetch := &stubKeyUsage{info: openrouter.KeyInfo{Usage: 40, UsageMonthly: 3.25}}
		svc := newSvc(t, fetch, &fakeClock{start})
		saveKey(t, svc, "openrouter", "k")

		got, err := svc.AIUsage(t.Context(), "user-1")
		if err != nil {
			t.Fatalf("AIUsage() err = %v", err)
		}
		if got.Used == nil || *got.Used != 3.25 || got.Limit != nil || got.Percent != nil || got.Level != dto.UsageOK {
			t.Errorf("AIUsage() = %+v, want used 3.25, no limit, no percent, level ok", got)
		}
	})

	t.Run("second request within 15 minutes makes no outbound call", func(t *testing.T) {
		fetch := &stubKeyUsage{info: openrouter.KeyInfo{Usage: 1, UsageMonthly: 1}}
		clock := &fakeClock{start}
		svc := newSvc(t, fetch, clock)
		saveKey(t, svc, "openrouter", "k")

		for _, advance := range []time.Duration{0, 14 * time.Minute} {
			clock.now = start.Add(advance)
			if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
				t.Fatalf("AIUsage() err = %v", err)
			}
		}
		if len(fetch.calls) != 1 {
			t.Fatalf("outbound calls = %d, want 1", len(fetch.calls))
		}

		clock.now = start.Add(15 * time.Minute)
		if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
			t.Fatalf("AIUsage() err = %v", err)
		}
		if len(fetch.calls) != 2 {
			t.Errorf("outbound calls after expiry = %d, want 2", len(fetch.calls))
		}
	})

	t.Run("saving or removing a key refreshes the usage", func(t *testing.T) {
		fetch := &stubKeyUsage{info: openrouter.KeyInfo{Usage: 1, UsageMonthly: 1}}
		svc := newSvc(t, fetch, &fakeClock{start})
		saveKey(t, svc, "openrouter", "old")
		if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
			t.Fatal(err)
		}

		saveKey(t, svc, "openrouter", "new")
		if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"old", "new"}, fetch.calls); diff != "" {
			t.Errorf("fetched keys (-want +got):\n%s", diff)
		}

		if _, err := svc.UpdateCredential(t.Context(), "user-1", dto.UpsertCredentialInput{Provider: "openrouter"}); err != nil {
			t.Fatal(err)
		}
		got, err := svc.AIUsage(t.Context(), "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != dto.QuotaNotConfigured {
			t.Errorf("AIUsage() after removal status = %q, want not_configured", got.Status)
		}
	})

	t.Run("a key change during a fetch does not cache the old key's usage", func(t *testing.T) {
		fetch := &stubKeyUsage{info: openrouter.KeyInfo{Usage: 1, UsageMonthly: 1}}
		svc := newSvc(t, fetch, &fakeClock{start})
		saveKey(t, svc, "openrouter", "old")
		fetch.onCall = func() { saveKey(t, svc, "openrouter", "new") }
		if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
			t.Fatal(err)
		}

		fetch.onCall = nil
		if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"old", "new"}, fetch.calls); diff != "" {
			t.Errorf("fetched keys (-want +got):\n%s", diff)
		}
	})

	t.Run("one user never sees another's usage", func(t *testing.T) {
		fetch := &stubKeyUsage{info: openrouter.KeyInfo{Usage: 1, UsageMonthly: 1}}
		svc := newSvc(t, fetch, &fakeClock{start})
		saveKey(t, svc, "openrouter", "k")
		if _, err := svc.AIUsage(t.Context(), "user-1"); err != nil {
			t.Fatal(err)
		}

		got, err := svc.AIUsage(t.Context(), "user-2")
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != dto.QuotaNotConfigured || got.Used != nil {
			t.Errorf("AIUsage(user-2) = %+v, want not_configured with no usage", got)
		}
	})

	t.Run("fetch failure is an error row and is not cached", func(t *testing.T) {
		fetch := &stubKeyUsage{err: errors.New("boom")}
		svc := newSvc(t, fetch, &fakeClock{start})
		saveKey(t, svc, "openrouter", "k")

		for range 2 {
			got, err := svc.AIUsage(t.Context(), "user-1")
			if err != nil {
				t.Fatalf("AIUsage() err = %v", err)
			}
			if got.Status != dto.QuotaError || got.Level != dto.UsageOK || got.Error == "" {
				t.Errorf("AIUsage() = %+v, want error status, ok level, message", got)
			}
		}
		if len(fetch.calls) != 2 {
			t.Errorf("outbound calls = %d, want 2", len(fetch.calls))
		}
	})
}
