package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

type failingCreateUserTx struct {
	identity.Store
	err error
}

func (f failingCreateUserTx) CreateUserTx(context.Context, pgx.Tx, string, string, string) (dto.User, error) {
	return dto.User{}, f.err
}

type seeder struct {
	seededUserID string
	err          error
}

func (s *seeder) SeedDefaults(_ context.Context, _ pgx.Tx, userID string) error {
	s.seededUserID = userID
	return s.err
}

func newService(t *testing.T, st identity.Store, sd identity.StatusSeeder) *identity.Service {
	t.Helper()
	return identity.NewService(identity.Deps{Store: st, Seeder: sd, Cipher: identitytest.NewCipher(t)})
}

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
	if _, err := st.CreateUser(context.Background(), username, string(hash), ""); err != nil {
		t.Fatal(err)
	}
}

func TestLoginValidCredentials(t *testing.T) {
	st := identitytest.NewFakeStore()
	seedUser(t, st, "alice", "secret")

	session, user, err := newService(t, st, &seeder{}).Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if session.UserID != user.ID || user.Username != "alice" {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	st := identitytest.NewFakeStore()
	seedUser(t, st, "alice", "secret")

	_, _, err := newService(t, st, &seeder{}).Login(context.Background(), "alice", "wrong")
	if !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Fatalf("err = %v, want KindUnauthorized", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	_, _, err := newService(t, identitytest.NewFakeStore(), &seeder{}).Login(context.Background(), "nobody", "secret")
	if !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Fatalf("err = %v, want KindUnauthorized", err)
	}
}

func TestSignupSucceeds(t *testing.T) {
	session, user, err := newService(t, identitytest.NewFakeStore(), &seeder{}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "bob" || session.UserID != user.ID {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestSignupSeedsDefaultStatusesForTheNewUser(t *testing.T) {
	sd := &seeder{}
	_, user, err := newService(t, identitytest.NewFakeStore(), sd).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if sd.seededUserID != user.ID {
		t.Fatalf("seededUserID = %q, want %q", sd.seededUserID, user.ID)
	}
}

func TestSignupFailsWhenSeedingFails(t *testing.T) {
	st := identitytest.NewFakeStore()
	seedErr := errors.New("seed boom")

	session, _, err := newService(t, st, &seeder{err: seedErr}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if !errors.Is(err, seedErr) {
		t.Fatalf("err = %v, want %v", err, seedErr)
	}
	if session != (dto.Session{}) {
		t.Errorf("session = %+v, want none created when seeding fails", session)
	}
}

func TestSignupRejectsMissingCredentials(t *testing.T) {
	_, _, err := newService(t, identitytest.NewFakeStore(), &seeder{}).Signup(context.Background(), "", "", "")
	if !apperr.IsKind(err, apperr.KindInvalid) {
		t.Fatalf("err = %v, want KindInvalid", err)
	}
}

func TestSignupDuplicateUsername(t *testing.T) {
	svc := newService(t, failingCreateUserTx{Store: identitytest.NewFakeStore(), err: store.ErrUsernameTaken}, &seeder{})

	_, _, err := svc.Signup(context.Background(), "alice", "pass", "")
	if !apperr.IsKind(err, apperr.KindConflict) {
		t.Fatalf("err = %v, want KindConflict", err)
	}
}

func TestLogoutDeletesTheSession(t *testing.T) {
	st := identitytest.NewFakeStore()
	user, err := st.CreateUser(context.Background(), "dana", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateSession(context.Background(), user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if err := newService(t, st, &seeder{}).Logout(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetSession(context.Background(), session.ID); err == nil {
		t.Fatal("want session gone after logout")
	}
}

func TestUpdateCredentialRequiresProvider(t *testing.T) {
	_, err := newService(t, identitytest.NewFakeStore(), &seeder{}).UpdateCredential(context.Background(), "user-1", dto.UpsertCredentialInput{})
	if !apperr.IsKind(err, apperr.KindInvalid) {
		t.Fatalf("err = %v, want KindInvalid", err)
	}
}

func TestUpdateCredentialSavesAQuotedKeyTrimmedAndGetCredentialDecrypts(t *testing.T) {
	svc := newService(t, identitytest.NewFakeStore(), &seeder{})
	key := `"sk-test"`
	if _, err := svc.UpdateCredential(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic", APIKey: &key}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetCredential(context.Background(), "user-1", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-test" {
		t.Fatalf("GetCredential(...) = %q, want sk-test", got)
	}
}

func TestUpdateCredentialDeletesWhenKeyIsNil(t *testing.T) {
	svc := newService(t, identitytest.NewFakeStore(), &seeder{})
	key := `"sk-test"`
	if _, err := svc.UpdateCredential(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic", APIKey: &key}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateCredential(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "anthropic"}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.GetAIPrefs(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ConfiguredProviders) != 0 {
		t.Fatalf("ConfiguredProviders after delete = %v, want none", got.ConfiguredProviders)
	}
}

func TestGetCredentialNotFound(t *testing.T) {
	_, err := newService(t, identitytest.NewFakeStore(), &seeder{}).GetCredential(context.Background(), "user-1", "anthropic")
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetAIPrefs(t *testing.T) {
	t.Run("no configured providers", func(t *testing.T) {
		got, err := newService(t, identitytest.NewFakeStore(), &seeder{}).GetAIPrefs(context.Background(), "user-1")
		if err != nil {
			t.Fatal(err)
		}
		want := dto.AIPrefsView{ConfiguredProviders: []string{}, ScoringEnabled: false}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetAIPrefs(...) mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("configured provider enables scoring", func(t *testing.T) {
		svc := newService(t, identitytest.NewFakeStore(), &seeder{})
		key := "sk-test"
		if _, err := svc.UpdateCredential(context.Background(), "user-1", dto.UpsertCredentialInput{Provider: "openrouter", APIKey: &key}); err != nil {
			t.Fatal(err)
		}
		got, err := svc.GetAIPrefs(context.Background(), "user-1")
		if err != nil {
			t.Fatal(err)
		}
		want := dto.AIPrefsView{ConfiguredProviders: []string{"openrouter"}, ScoringEnabled: true}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetAIPrefs(...) mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("credential list error propagates", func(t *testing.T) {
		listErr := errors.New("creds down")
		svc := newService(t, failingList{Store: identitytest.NewFakeStore(), err: listErr}, &seeder{})
		if _, err := svc.GetAIPrefs(context.Background(), "user-1"); !errors.Is(err, listErr) {
			t.Fatalf("err = %v, want %v", err, listErr)
		}
	})
}

func TestProfile(t *testing.T) {
	st := identitytest.NewFakeStore()
	svc := newService(t, st, &seeder{})
	user, err := st.CreateUser(context.Background(), "alice", "hash", "old@example.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.UpdateProfile(context.Background(), user.ID, dto.UpdateProfileInput{Email: "new@example.com"}); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetProfile(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := dto.Profile{Username: "alice", Email: "new@example.com"}
	if got != want {
		t.Fatalf("GetProfile(...) = %+v, want %+v", got, want)
	}
}
