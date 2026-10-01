package identity_test

import (
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func newGoogleService(t *testing.T, gc *identitytest.DocsClient) *identity.Service {
	t.Helper()
	return identity.NewService(testDeps(t, identity.Deps{GoogleClient: gc}))
}

func TestGoogleStatus(t *testing.T) {
	ctx := t.Context()

	t.Run("no token reads as disconnected", func(t *testing.T) {
		got, err := newGoogleService(t, identitytest.Unlinked(false)).GoogleStatus(ctx, "user-1")
		if err != nil || got.Connected {
			t.Fatalf("GoogleStatus(...) = %+v, %v, want disconnected", got, err)
		}
	})

	t.Run("unusable token reads as disconnected", func(t *testing.T) {
		got, err := newGoogleService(t, identitytest.Disconnected(google.ErrTokenUnusable)).GoogleStatus(ctx, "user-1")
		if err != nil || got.Connected {
			t.Fatalf("GoogleStatus(...) = %+v, %v, want disconnected", got, err)
		}
	})

	t.Run("other client errors propagate", func(t *testing.T) {
		want := errors.New("boom")
		if _, err := newGoogleService(t, identitytest.Disconnected(want)).GoogleStatus(ctx, "user-1"); !errors.Is(err, want) {
			t.Fatalf("err = %v, want %v", err, want)
		}
	})

	t.Run("CanWrite and CanEditDocs reflect the granted scopes", func(t *testing.T) {
		for _, granted := range []bool{false, true} {
			svc := newGoogleService(t, identitytest.Unlinked(granted))
			if err := svc.Connect(ctx, "user-1", "code"); err != nil {
				t.Fatal(err)
			}
			got, err := svc.GoogleStatus(ctx, "user-1")
			if err != nil {
				t.Fatal(err)
			}
			if !got.Connected || got.Email != "user-1@example.com" || got.CanWrite != granted || got.CanEditDocs != granted {
				t.Errorf("granted=%v: GoogleStatus(...) = %+v", granted, got)
			}
		}
	})
}

func TestConnectPropagatesExchangeError(t *testing.T) {
	want := errors.New("exchange failed")
	gc := identitytest.Unlinked(false)
	gc.ExchangeErr = want
	if err := newGoogleService(t, gc).Connect(t.Context(), "user-1", "code"); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}
