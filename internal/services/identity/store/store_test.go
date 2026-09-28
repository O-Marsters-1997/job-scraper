package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	return store.New(pgtest.New(t))
}

func TestIdentityStoreContract(t *testing.T) {
	identitytest.RunStoreContract(t, func(t *testing.T) identity.Store {
		t.Helper()
		return newStore(t)
	})
}

func TestSignupSeedingFailureLeavesNoUserRow(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	tx, err := st.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	if _, err := st.CreateUserTx(ctx, tx, "frank", "hash", ""); err != nil {
		t.Fatalf("CreateUserTx: %v", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO application_statuses (user_id, name, colour) VALUES ($1, 'Draft', '#64748b')`,
		"00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("want seeding insert to fail")
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if _, err := st.GetUserByUsername(ctx, "frank"); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound after rollback", err)
	}
}

func TestUpdateEmail(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "frank", "hash", "frank@example.com")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	updated, err := st.UpdateEmail(ctx, user.ID, "new@example.com")
	if err != nil {
		t.Fatalf("UpdateEmail: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("email = %q, want new@example.com", updated.Email)
	}

	got, err := st.GetProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetProfile after update: %v", err)
	}
	if got.Email != "new@example.com" {
		t.Errorf("email = %q, want new@example.com", got.Email)
	}
}

func TestUserAICredentials(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "grace", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := st.GetUserAICredential(ctx, user.ID, "anthropic"); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound before any credential saved", err)
	}

	if err := st.UpsertUserAICredential(ctx, user.ID, "anthropic", "enc-1"); err != nil {
		t.Fatalf("UpsertUserAICredential: %v", err)
	}
	got, err := st.GetUserAICredential(ctx, user.ID, "anthropic")
	if err != nil {
		t.Fatalf("GetUserAICredential: %v", err)
	}
	if got != "enc-1" {
		t.Errorf("got %q, want enc-1", got)
	}

	if err := st.UpsertUserAICredential(ctx, user.ID, "anthropic", "enc-2"); err != nil {
		t.Fatalf("UpsertUserAICredential (update): %v", err)
	}
	got, err = st.GetUserAICredential(ctx, user.ID, "anthropic")
	if err != nil {
		t.Fatalf("GetUserAICredential after update: %v", err)
	}
	if got != "enc-2" {
		t.Errorf("got %q, want enc-2", got)
	}

	if err := st.UpsertUserAICredential(ctx, user.ID, "openrouter", "enc-3"); err != nil {
		t.Fatalf("UpsertUserAICredential (second provider): %v", err)
	}
	providers, err := st.ListUserAICredentialProviders(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListUserAICredentialProviders: %v", err)
	}
	if len(providers) != 2 || providers[0] != "anthropic" || providers[1] != "openrouter" {
		t.Errorf("providers = %v, want [anthropic openrouter]", providers)
	}

	if err := st.DeleteUserAICredential(ctx, user.ID, "anthropic"); err != nil {
		t.Fatalf("DeleteUserAICredential: %v", err)
	}
	if _, err := st.GetUserAICredential(ctx, user.ID, "anthropic"); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound after delete", err)
	}
}

func TestGoogleToken(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "henry", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := st.GetGoogleToken(ctx, user.ID); !errors.Is(err, google.ErrTokenNotFound) {
		t.Fatalf("err = %v, want google.ErrTokenNotFound", err)
	}

	expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	if err := st.UpsertGoogleToken(ctx, dto.UpsertGoogleTokenInput{
		UserID:          user.ID,
		AccessTokenEnc:  "access-enc-1",
		RefreshTokenEnc: "refresh-enc-1",
		TokenType:       "Bearer",
		Expiry:          expiry,
		Scope:           "https://www.googleapis.com/auth/drive.readonly",
	}); err != nil {
		t.Fatalf("UpsertGoogleToken: %v", err)
	}

	got, err := st.GetGoogleToken(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetGoogleToken: %v", err)
	}
	if got.AccessTokenEnc != "access-enc-1" || got.RefreshTokenEnc != "refresh-enc-1" {
		t.Errorf("got = %+v", got)
	}
	if !got.Expiry.Equal(expiry) {
		t.Errorf("expiry = %v, want %v", got.Expiry, expiry)
	}

	if err := st.UpsertGoogleToken(ctx, dto.UpsertGoogleTokenInput{
		UserID:          user.ID,
		AccessTokenEnc:  "access-enc-2",
		RefreshTokenEnc: "refresh-enc-2",
		TokenType:       "Bearer",
		Scope:           "https://www.googleapis.com/auth/drive.readonly",
	}); err != nil {
		t.Fatalf("UpsertGoogleToken (update): %v", err)
	}
	got, err = st.GetGoogleToken(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetGoogleToken after update: %v", err)
	}
	if got.AccessTokenEnc != "access-enc-2" {
		t.Errorf("access token = %q, want access-enc-2", got.AccessTokenEnc)
	}

	if err := st.DeleteGoogleToken(ctx, user.ID); err != nil {
		t.Fatalf("DeleteGoogleToken: %v", err)
	}
	if _, err := st.GetGoogleToken(ctx, user.ID); !errors.Is(err, google.ErrTokenNotFound) {
		t.Fatalf("err = %v, want google.ErrTokenNotFound after delete", err)
	}
}
