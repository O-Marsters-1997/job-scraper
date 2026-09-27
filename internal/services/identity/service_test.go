package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

type fakeTx struct {
	committed  bool
	rolledBack bool
}

func (t *fakeTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }
func (t *fakeTx) Commit(context.Context) error          { t.committed = true; return nil }
func (t *fakeTx) Rollback(context.Context) error        { t.rolledBack = true; return nil }
func (t *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *fakeTx) Conn() *pgx.Conn                                         { return nil }

type fakeStore struct {
	usersByName map[string]dto.User
	sessions    map[string]dto.Session
	createErr   error
	tx          *fakeTx
}

func newFakeStore() *fakeStore {
	return &fakeStore{usersByName: map[string]dto.User{}, sessions: map[string]dto.Session{}, tx: &fakeTx{}}
}

func (f *fakeStore) seed(u dto.User) { f.usersByName[u.Username] = u }

func (f *fakeStore) GetUserByUsername(_ context.Context, username string) (dto.User, error) {
	u, ok := f.usersByName[username]
	if !ok {
		return dto.User{}, errors.New("not found")
	}
	return u, nil
}

func (f *fakeStore) Begin(context.Context) (pgx.Tx, error) { return f.tx, nil }

func (f *fakeStore) CreateUserTx(_ context.Context, _ pgx.Tx, username, passwordHash, email string) (dto.User, error) {
	if f.createErr != nil {
		return dto.User{}, f.createErr
	}
	u := dto.User{ID: "user-" + username, Username: username, PasswordHash: passwordHash, Email: email}
	f.usersByName[username] = u
	return u, nil
}

func (f *fakeStore) CreateSession(_ context.Context, userID string, expiresAt time.Time) (dto.Session, error) {
	s := dto.Session{ID: "session-" + userID, UserID: userID, ExpiresAt: expiresAt}
	f.sessions[s.ID] = s
	return s, nil
}

func (f *fakeStore) DeleteSession(_ context.Context, id string) error {
	delete(f.sessions, id)
	return nil
}

type fakeSeeder struct {
	seededUserID string
	err          error
}

func (f *fakeSeeder) SeedDefaults(_ context.Context, _ pgx.Tx, userID string) error {
	f.seededUserID = userID
	return f.err
}

func TestLoginValidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	st := newFakeStore()
	st.seed(dto.User{ID: "user-1", Username: "alice", PasswordHash: string(hash)})

	session, user, err := identity.NewService(st, &fakeSeeder{}).Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if session.UserID != "user-1" || user.Username != "alice" {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	st := newFakeStore()
	st.seed(dto.User{ID: "user-1", Username: "alice", PasswordHash: string(hash)})

	_, _, err := identity.NewService(st, &fakeSeeder{}).Login(context.Background(), "alice", "wrong")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnauthorized.Status() {
		t.Fatalf("status = %v, ok = %v, want 401", status, ok)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	_, _, err := identity.NewService(newFakeStore(), &fakeSeeder{}).Login(context.Background(), "nobody", "secret")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnauthorized.Status() {
		t.Fatalf("status = %v, ok = %v, want 401", status, ok)
	}
}

func TestSignupSucceeds(t *testing.T) {
	session, user, err := identity.NewService(newFakeStore(), &fakeSeeder{}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "bob" || session.UserID != user.ID {
		t.Fatalf("session = %+v, user = %+v", session, user)
	}
}

func TestSignupSeedsDefaultStatusesForTheNewUser(t *testing.T) {
	seeder := &fakeSeeder{}
	_, user, err := identity.NewService(newFakeStore(), seeder).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if seeder.seededUserID != user.ID {
		t.Fatalf("seededUserID = %q, want %q", seeder.seededUserID, user.ID)
	}
}

func TestSignupRollsBackAndFailsWhenSeedingFails(t *testing.T) {
	st := newFakeStore()
	seedErr := errors.New("seed boom")

	_, _, err := identity.NewService(st, &fakeSeeder{err: seedErr}).Signup(context.Background(), "bob", "hunter2", "bob@example.com")
	if !errors.Is(err, seedErr) {
		t.Fatalf("err = %v, want %v", err, seedErr)
	}
	if st.tx.committed {
		t.Error("want tx not committed when seeding fails")
	}
	if !st.tx.rolledBack {
		t.Error("want tx rolled back when seeding fails")
	}
	if len(st.sessions) != 0 {
		t.Errorf("sessions = %+v, want none created when seeding fails", st.sessions)
	}
}

func TestSignupRejectsMissingCredentials(t *testing.T) {
	_, _, err := identity.NewService(newFakeStore(), &fakeSeeder{}).Signup(context.Background(), "", "", "")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestSignupDuplicateUsername(t *testing.T) {
	st := newFakeStore()
	st.createErr = store.ErrUsernameTaken

	_, _, err := identity.NewService(st, &fakeSeeder{}).Signup(context.Background(), "alice", "pass", "")
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindConflict.Status() {
		t.Fatalf("status = %v, ok = %v, want 409", status, ok)
	}
}

func TestLogoutDeletesTheSession(t *testing.T) {
	st := newFakeStore()
	st.sessions["session-abc"] = dto.Session{ID: "session-abc", UserID: "user-1"}

	if err := identity.NewService(st, &fakeSeeder{}).Logout(context.Background(), "session-abc"); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.sessions["session-abc"]; ok {
		t.Fatal("want session gone after logout")
	}
}
