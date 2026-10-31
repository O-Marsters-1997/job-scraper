package cvtailortest

import (
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

func runOutreachHookContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	t.Run("a new hook replaces the text and adds to the cost", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := f.SeedContact(t, f.UserID)
		if text, cost, err := f.Store.OutreachHook(ctx, f.UserID, id); err != nil || text != "" || cost != 0 {
			t.Fatalf("OutreachHook() before = %q, %v, %v, want empty", text, cost, err)
		}
		for _, text := range []string{"first", "second"} {
			if err := f.Store.SaveOutreachHook(ctx, f.UserID, id, text, "m", 0.5); err != nil {
				t.Fatalf("SaveOutreachHook(%s): %v", text, err)
			}
		}
		text, cost, err := f.Store.OutreachHook(ctx, f.UserID, id)
		if err != nil || text != "second" || cost != 1 {
			t.Errorf("OutreachHook() = %q, %v, %v, want second, 1", text, cost, err)
		}
	})

	t.Run("another user's or a missing contact is not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := f.SeedContact(t, f.UserID)
		for name, c := range map[string][2]string{"other user": {f.Other, id}, "missing": {f.UserID, missingID}} {
			err := f.Store.SaveOutreachHook(ctx, c[0], c[1], "x", "m", 0)
			if !errors.Is(err, store.ErrContactNotFound) {
				t.Errorf("%s: SaveOutreachHook() error = %v, want ErrContactNotFound", name, err)
			}
		}
	})
}
