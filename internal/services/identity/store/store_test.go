package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"

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

func TestCreateUserTxCommits(t *testing.T) {
	pool := pgtest.New(t)
	st := store.New(pool)

	pgtest.InTx(t, pool, true, func(tx pgx.Tx) error {
		_, err := st.CreateUserTx(t.Context(), tx, "carol", "hash", "")
		return err
	})

	if _, err := st.GetUserByUsername(t.Context(), "carol"); err != nil {
		t.Errorf("GetUserByUsername(carol) err = %v, want user after commit", err)
	}
}

func TestSignupSeedingFailureLeavesNoUserRow(t *testing.T) {
	pool := pgtest.New(t)
	st := store.New(pool)
	ctx := t.Context()

	var seedErr error
	pgtest.InTx(t, pool, false, func(tx pgx.Tx) error {
		if _, err := st.CreateUserTx(ctx, tx, "frank", "hash", ""); err != nil {
			return err
		}
		_, seedErr = tx.Exec(ctx, `INSERT INTO application_statuses (user_id, name, colour) VALUES ($1, 'Draft', '#64748b')`,
			"00000000-0000-0000-0000-000000000000")
		return nil
	})
	if seedErr == nil {
		t.Fatal("want seeding insert to fail")
	}

	if _, err := st.GetUserByUsername(ctx, "frank"); !errors.Is(err, data.ErrNotFound) {
		t.Errorf("GetUserByUsername(frank) err = %v, want ErrNotFound after rollback", err)
	}
}

func TestGoogleToken(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	user, err := st.CreateUser(ctx, "henry", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := st.GetGoogleToken(ctx, user.ID); !errors.Is(err, google.ErrTokenNotFound) {
		t.Fatalf("GetGoogleToken before upsert err = %v, want google.ErrTokenNotFound", err)
	}

	scope := "https://www.googleapis.com/auth/drive.readonly"
	expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	first := dto.UpsertGoogleTokenInput{
		UserID: user.ID, AccessTokenEnc: "access-enc-1", RefreshTokenEnc: "refresh-enc-1",
		TokenType: "Bearer", Expiry: expiry, Scope: scope,
	}
	if err := st.UpsertGoogleToken(ctx, first); err != nil {
		t.Fatalf("UpsertGoogleToken: %v", err)
	}
	got, err := st.GetGoogleToken(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetGoogleToken: %v", err)
	}
	want := dto.GoogleToken{AccessTokenEnc: "access-enc-1", RefreshTokenEnc: "refresh-enc-1", TokenType: "Bearer", Expiry: expiry, Scope: scope}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetGoogleToken mismatch (-want +got):\n%s", diff)
	}

	second := dto.UpsertGoogleTokenInput{
		UserID: user.ID, AccessTokenEnc: "access-enc-2", RefreshTokenEnc: "refresh-enc-2",
		TokenType: "Bearer", Scope: scope,
	}
	if err := st.UpsertGoogleToken(ctx, second); err != nil {
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
		t.Errorf("GetGoogleToken after delete err = %v, want google.ErrTokenNotFound", err)
	}
}
