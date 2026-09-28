package identitytest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

// RunStoreContract proves newStore's identity.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) identity.Store) {
	t.Helper()

	t.Run("create user returns the user", func(t *testing.T) {
		st := newStore(t)
		got, err := st.CreateUser(context.Background(), "alice", "hash", "alice@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if got.ID == "" || got.Username != "alice" || got.Email != "alice@example.com" {
			t.Fatalf("CreateUser(...) = %+v", got)
		}
	})

	t.Run("create user rejects duplicate username", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.CreateUser(context.Background(), "bob", "hash", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateUser(context.Background(), "bob", "hash2", ""); !errors.Is(err, store.ErrUsernameTaken) {
			t.Fatalf("err = %v, want ErrUsernameTaken", err)
		}
	})

	t.Run("get user by username returns not found when missing", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.GetUserByUsername(context.Background(), "nobody"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("create user tx commits like create user", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		tx, err := st.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateUserTx(ctx, tx, "carol", "hash", ""); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetUserByUsername(ctx, "carol"); err != nil {
			t.Fatalf("GetUserByUsername after commit: %v", err)
		}
	})

	t.Run("create session then get session returns it", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "dana", "hash", "")
		if err != nil {
			t.Fatal(err)
		}
		session, err := st.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		got, err := st.GetSession(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.UserID != user.ID || got.Username != "dana" {
			t.Fatalf("GetSession(...) = %+v", got)
		}
	})

	t.Run("get session returns not found when expired", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "erin", "hash", "")
		if err != nil {
			t.Fatal(err)
		}
		session, err := st.CreateSession(ctx, user.ID, time.Now().Add(-time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetSession(ctx, session.ID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete session removes it", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "frank", "hash", "")
		if err != nil {
			t.Fatal(err)
		}
		session, err := st.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteSession(ctx, session.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetSession(ctx, session.ID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete expired sessions leaves valid ones", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "grace", "hash", "")
		if err != nil {
			t.Fatal(err)
		}
		valid, err := st.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateSession(ctx, user.ID, time.Now().Add(-time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteExpiredSessions(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetSession(ctx, valid.ID); err != nil {
			t.Errorf("valid session should survive cleanup: %v", err)
		}
	})

	t.Run("get profile returns username and email", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "henry", "hash", "henry@example.com")
		if err != nil {
			t.Fatal(err)
		}
		got, err := st.GetProfile(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Username != "henry" || got.Email != "henry@example.com" {
			t.Fatalf("GetProfile(...) = %+v", got)
		}
	})

	t.Run("get profile returns not found when missing", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.GetProfile(context.Background(), missingID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("update email changes the profile", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "ivy", "hash", "ivy@example.com")
		if err != nil {
			t.Fatal(err)
		}
		updated, err := st.UpdateEmail(ctx, user.ID, "new@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if updated.Email != "new@example.com" {
			t.Errorf("UpdateEmail(...).Email = %q, want new@example.com", updated.Email)
		}
		got, err := st.GetProfile(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Email != "new@example.com" {
			t.Errorf("GetProfile(...).Email = %q, want new@example.com", got.Email)
		}
	})

	t.Run("update email returns not found when missing", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.UpdateEmail(context.Background(), missingID, "a@example.com"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ai credentials upsert, list and delete", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		user, err := st.CreateUser(ctx, "judy", "hash", "")
		if err != nil {
			t.Fatal(err)
		}

		if _, err := st.GetUserAICredential(ctx, user.ID, "anthropic"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound before any credential saved", err)
		}

		if err := st.UpsertUserAICredential(ctx, user.ID, "anthropic", "enc-1"); err != nil {
			t.Fatal(err)
		}
		got, err := st.GetUserAICredential(ctx, user.ID, "anthropic")
		if err != nil {
			t.Fatal(err)
		}
		if got != "enc-1" {
			t.Errorf("GetUserAICredential(...) = %q, want enc-1", got)
		}

		if err := st.UpsertUserAICredential(ctx, user.ID, "anthropic", "enc-2"); err != nil {
			t.Fatal(err)
		}
		got, err = st.GetUserAICredential(ctx, user.ID, "anthropic")
		if err != nil {
			t.Fatal(err)
		}
		if got != "enc-2" {
			t.Errorf("GetUserAICredential(...) after update = %q, want enc-2", got)
		}

		if err := st.UpsertUserAICredential(ctx, user.ID, "openrouter", "enc-3"); err != nil {
			t.Fatal(err)
		}
		providers, err := st.ListUserAICredentialProviders(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"anthropic", "openrouter"}; !slices.Equal(providers, want) {
			t.Errorf("ListUserAICredentialProviders(...) = %v, want %v", providers, want)
		}

		if err := st.DeleteUserAICredential(ctx, user.ID, "anthropic"); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetUserAICredential(ctx, user.ID, "anthropic"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound after delete", err)
		}
	})
}

const missingID = "00000000-0000-0000-0000-000000000000"
