package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	return store.New(pgtest.New(t))
}

func TestCreateUser(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	created, err := st.CreateUser(ctx, "alice", "hashed-password", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.Username != "alice" {
		t.Errorf("username = %q, want alice", created.Username)
	}
	if created.PasswordHash != "hashed-password" {
		t.Errorf("password_hash = %q, want hashed-password", created.PasswordHash)
	}
	if created.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestCreateUserDuplicateUsername(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	if _, err := st.CreateUser(ctx, "bob", "hash", ""); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	_, err := st.CreateUser(ctx, "bob", "hash2", "")
	if !errors.Is(err, store.ErrUsernameTaken) {
		t.Fatalf("err = %v, want ErrUsernameTaken", err)
	}
}

func TestGetUserByUsername(t *testing.T) {
	t.Run("returns created user", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		created, err := st.CreateUser(ctx, "alice", "hashed-password", "")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		got, err := st.GetUserByUsername(ctx, "alice")
		if err != nil {
			t.Fatalf("GetUserByUsername: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("id = %q, want %q", got.ID, created.ID)
		}
	})

	t.Run("returns ErrNotFound when missing", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		_, err := st.GetUserByUsername(ctx, "nonexistent")
		if !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestCreateSession(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "bob", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	session, err := st.CreateSession(ctx, user.ID, time.Now().Add(30*24*time.Hour).UTC())
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
	if session.UserID != user.ID {
		t.Errorf("user_id = %q, want %q", session.UserID, user.ID)
	}
}

func TestGetSession(t *testing.T) {
	t.Run("returns session", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		user, err := st.CreateUser(ctx, "bob", "hash", "")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		session, err := st.CreateSession(ctx, user.ID, time.Now().Add(30*24*time.Hour).UTC())
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		got, err := st.GetSession(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got.Username != "bob" {
			t.Errorf("username = %q, want bob", got.Username)
		}
	})

	t.Run("returns ErrNotFound when expired", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		user, err := st.CreateUser(ctx, "charlie", "hash", "")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		session, err := st.CreateSession(ctx, user.ID, time.Now().Add(-time.Second))
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		_, err = st.GetSession(ctx, session.ID)
		if !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestDeleteSession(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "dana", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	session, err := st.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := st.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	_, err = st.GetSession(ctx, session.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "eve", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	valid, err := st.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession (valid): %v", err)
	}
	if _, err := st.CreateSession(ctx, user.ID, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("CreateSession (expired): %v", err)
	}

	if err := st.DeleteExpiredSessions(ctx); err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}

	if _, err := st.GetSession(ctx, valid.ID); err != nil {
		t.Errorf("valid session should survive cleanup: %v", err)
	}
}
