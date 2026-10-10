package identitytest

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

// RunStoreContract proves newStore's identity.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) identity.Store) {
	t.Helper()

	t.Run("create user returns the user", func(t *testing.T) {
		st := newStore(t)
		got, err := st.CreateUser(t.Context(), "alice", "hash", "alice@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if got.ID == "" || got.Username != "alice" || got.Email != "alice@example.com" {
			t.Fatalf("CreateUser(...) = %+v", got)
		}
	})

	t.Run("create user rejects duplicate username", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.CreateUser(t.Context(), "bob", "hash", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateUser(t.Context(), "bob", "hash2", ""); !errors.Is(err, store.ErrUsernameTaken) {
			t.Fatalf("err = %v, want ErrUsernameTaken", err)
		}
	})

	t.Run("get user by username returns not found when missing", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.GetUserByUsername(t.Context(), "nobody"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("create session then get session returns it", func(t *testing.T) {
		st := newStore(t)
		user, session := newSession(t, st, "dana", time.Hour)
		got, err := st.GetSession(t.Context(), session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.UserID != user.ID || got.Username != "dana" {
			t.Fatalf("GetSession(...) = %+v", got)
		}
	})

	t.Run("new user is a user and SetRole makes them admin", func(t *testing.T) {
		st := newStore(t)
		_, session := newSession(t, st, "olly", time.Hour)
		got, err := st.GetSession(t.Context(), session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Role != dto.RoleUser {
			t.Errorf("GetSession(new user).Role = %q, want %q", got.Role, dto.RoleUser)
		}
		if err := st.SetRole(t.Context(), "olly", dto.RoleAdmin); err != nil {
			t.Fatal(err)
		}
		got, err = st.GetSession(t.Context(), session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Role != dto.RoleAdmin {
			t.Errorf("GetSession after SetRole.Role = %q, want %q", got.Role, dto.RoleAdmin)
		}
	})

	t.Run("set role returns not found for an unknown username", func(t *testing.T) {
		st := newStore(t)
		if err := st.SetRole(t.Context(), "nobody", dto.RoleAdmin); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("SetRole(nobody) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("set password hash replaces the stored hash", func(t *testing.T) {
		st := newStore(t)
		if _, err := st.CreateUser(t.Context(), "gina", "old-hash", ""); err != nil {
			t.Fatal(err)
		}
		if err := st.SetPasswordHash(t.Context(), "gina", "new-hash"); err != nil {
			t.Fatal(err)
		}
		got, err := st.GetUserByUsername(t.Context(), "gina")
		if err != nil {
			t.Fatal(err)
		}
		if got.PasswordHash != "new-hash" {
			t.Errorf("GetUserByUsername(gina).PasswordHash = %q, want new-hash", got.PasswordHash)
		}
	})

	t.Run("set password hash returns not found for an unknown username", func(t *testing.T) {
		st := newStore(t)
		if err := st.SetPasswordHash(t.Context(), "nobody", "hash"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("SetPasswordHash(nobody) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get session returns not found when expired", func(t *testing.T) {
		st := newStore(t)
		_, session := newSession(t, st, "erin", -time.Second)
		if _, err := st.GetSession(t.Context(), session.ID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete session removes it", func(t *testing.T) {
		st := newStore(t)
		_, session := newSession(t, st, "frank", time.Hour)
		if err := st.DeleteSession(t.Context(), session.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetSession(t.Context(), session.ID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete expired sessions leaves valid ones", func(t *testing.T) {
		st := newStore(t)
		user, valid := newSession(t, st, "grace", time.Hour)
		if _, err := st.CreateSession(t.Context(), user.ID, time.Now().Add(-time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteExpiredSessions(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetSession(t.Context(), valid.ID); err != nil {
			t.Errorf("valid session should survive cleanup: %v", err)
		}
	})

	t.Run("get profile returns username and email", func(t *testing.T) {
		st := newStore(t)
		ctx := t.Context()
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
		if _, err := st.GetProfile(t.Context(), missingID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("update email changes the profile", func(t *testing.T) {
		st := newStore(t)
		ctx := t.Context()
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
		if _, err := st.UpdateEmail(t.Context(), missingID, "a@example.com"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get ai credential returns not found before any is saved", func(t *testing.T) {
		st, user := newUser(t, newStore(t), "judy")
		if _, err := st.GetUserAICredential(t.Context(), user.ID, "anthropic"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("upsert ai credential overwrites the key", func(t *testing.T) {
		st, user := newUser(t, newStore(t), "judy")
		for _, enc := range []string{"enc-1", "enc-2"} {
			if err := st.UpsertUserAICredential(t.Context(), user.ID, "anthropic", enc); err != nil {
				t.Fatal(err)
			}
		}
		got, err := st.GetUserAICredential(t.Context(), user.ID, "anthropic")
		if err != nil {
			t.Fatal(err)
		}
		if got != "enc-2" {
			t.Errorf("GetUserAICredential(...) = %q, want enc-2", got)
		}
	})

	t.Run("list ai credential providers returns every saved provider", func(t *testing.T) {
		st, user := newUser(t, newStore(t), "judy")
		for _, provider := range []string{"anthropic", "openrouter"} {
			if err := st.UpsertUserAICredential(t.Context(), user.ID, provider, "enc"); err != nil {
				t.Fatal(err)
			}
		}
		providers, err := st.ListUserAICredentialProviders(t.Context(), user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"anthropic", "openrouter"}; !slices.Equal(providers, want) {
			t.Errorf("ListUserAICredentialProviders(...) = %v, want %v", providers, want)
		}
	})

	t.Run("delete ai credential removes it", func(t *testing.T) {
		st, user := newUser(t, newStore(t), "judy")
		if err := st.UpsertUserAICredential(t.Context(), user.ID, "anthropic", "enc"); err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteUserAICredential(t.Context(), user.ID, "anthropic"); err != nil {
			t.Fatal(err)
		}
		if _, err := st.GetUserAICredential(t.Context(), user.ID, "anthropic"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func newUser(t *testing.T, st identity.Store, username string) (identity.Store, dto.User) {
	t.Helper()
	user, err := st.CreateUser(t.Context(), username, "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	return st, user
}

func newSession(t *testing.T, st identity.Store, username string, ttl time.Duration) (dto.User, dto.Session) {
	t.Helper()
	_, user := newUser(t, st, username)
	session, err := st.CreateSession(t.Context(), user.ID, time.Now().Add(ttl))
	if err != nil {
		t.Fatal(err)
	}
	return user, session
}

const missingID = "00000000-0000-0000-0000-000000000000"
