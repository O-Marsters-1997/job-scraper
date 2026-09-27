package auth_test

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/api/services/auth"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type store struct {
	*providers.MockUserProvider
	*providers.MockSessionProvider
}

func newStore() store {
	return store{
		providers.NewMockUserProvider(),
		providers.NewMockSessionProvider(),
	}
}

type fakeSeeder struct {
	seededUserID string
}

func (f *fakeSeeder) SeedDefaults(_ context.Context, userID string) error {
	f.seededUserID = userID
	return nil
}

func TestLoginValidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	st := newStore()
	st.MockUserProvider.Seed(dto.User{ID: "user-1", Username: "alice", PasswordHash: string(hash)})

	session, user, err := auth.New(st, &fakeSeeder{}).Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if session.UserID != "user-1" || user.Username != "alice" {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	st := newStore()
	st.MockUserProvider.Seed(dto.User{ID: "user-1", Username: "alice", PasswordHash: string(hash)})

	_, _, err := auth.New(st, &fakeSeeder{}).Login(context.Background(), "alice", "wrong")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnauthorized.Status() {
		t.Fatalf("status = %v, ok = %v, want 401", status, ok)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	_, _, err := auth.New(newStore(), &fakeSeeder{}).Login(context.Background(), "nobody", "secret")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnauthorized.Status() {
		t.Fatalf("status = %v, ok = %v, want 401", status, ok)
	}
}

func TestSignupSucceeds(t *testing.T) {
	session, user, err := auth.New(newStore(), &fakeSeeder{}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "bob" || session.UserID != user.ID {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestSignupSeedsDefaultStatusesForTheNewUser(t *testing.T) {
	seeder := &fakeSeeder{}
	_, user, err := auth.New(newStore(), seeder).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if seeder.seededUserID != user.ID {
		t.Fatalf("seededUserID = %q, want %q", seeder.seededUserID, user.ID)
	}
}

func TestSignupRejectsMissingCredentials(t *testing.T) {
	_, _, err := auth.New(newStore(), &fakeSeeder{}).Signup(context.Background(), "", "", "")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestSignupDuplicateUsername(t *testing.T) {
	st := newStore()
	st.MockUserProvider.CreateErr = providers.ErrUsernameTaken

	_, _, err := auth.New(st, &fakeSeeder{}).Signup(context.Background(), "alice", "pass", "")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindConflict.Status() {
		t.Fatalf("status = %v, ok = %v, want 409", status, ok)
	}
}

func TestLogoutDeletesTheSession(t *testing.T) {
	st := newStore()
	st.MockSessionProvider.Seed(dto.Session{ID: "session-abc", UserID: "user-1"})

	if err := auth.New(st, &fakeSeeder{}).Logout(context.Background(), "session-abc"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetSession(context.Background(), "session-abc"); err == nil {
		t.Fatal("want session gone after logout")
	}
}
