package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
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

	session, user, err := identity.NewService(st, &seeder{}).Login(context.Background(), "alice", "secret")
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

	_, _, err := identity.NewService(st, &seeder{}).Login(context.Background(), "alice", "wrong")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnauthorized.Status() {
		t.Fatalf("status = %v, ok = %v, want 401", status, ok)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	_, _, err := identity.NewService(identitytest.NewFakeStore(), &seeder{}).Login(context.Background(), "nobody", "secret")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnauthorized.Status() {
		t.Fatalf("status = %v, ok = %v, want 401", status, ok)
	}
}

func TestSignupSucceeds(t *testing.T) {
	session, user, err := identity.NewService(identitytest.NewFakeStore(), &seeder{}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "bob" || session.UserID != user.ID {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestSignupSeedsDefaultStatusesForTheNewUser(t *testing.T) {
	sd := &seeder{}
	_, user, err := identity.NewService(identitytest.NewFakeStore(), sd).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
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

	session, _, err := identity.NewService(st, &seeder{err: seedErr}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if !errors.Is(err, seedErr) {
		t.Fatalf("err = %v, want %v", err, seedErr)
	}
	if session != (dto.Session{}) {
		t.Errorf("session = %+v, want none created when seeding fails", session)
	}
}

func TestSignupRejectsMissingCredentials(t *testing.T) {
	_, _, err := identity.NewService(identitytest.NewFakeStore(), &seeder{}).Signup(context.Background(), "", "", "")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestSignupDuplicateUsername(t *testing.T) {
	svc := identity.NewService(failingCreateUserTx{Store: identitytest.NewFakeStore(), err: store.ErrUsernameTaken}, &seeder{})

	_, _, err := svc.Signup(context.Background(), "alice", "pass", "")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindConflict.Status() {
		t.Fatalf("status = %v, ok = %v, want 409", status, ok)
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

	if err := identity.NewService(st, &seeder{}).Logout(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetSession(context.Background(), session.ID); err == nil {
		t.Fatal("want session gone after logout")
	}
}
