package cvtailortest

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/promptparts"
)

func runPromptPartOverrideContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()
	const g = promptparts.ProfileRedraft

	t.Run("save upserts, is scoped to the user and delete removes it", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		first := promptparts.Override{Path: "voice.md", Text: "mine", BaseDefault: "default v1"}
		second := promptparts.Override{Path: "voice.md", Text: "mine again", BaseDefault: "default v2"}
		for _, o := range []promptparts.Override{first, second} {
			if err := f.Store.SavePartOverride(ctx, f.UserID, g, o); err != nil {
				t.Fatal(err)
			}
		}

		got, err := f.Store.ListPartOverrides(ctx, f.UserID, g)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]promptparts.Override{second}, got); diff != "" {
			t.Fatalf("ListPartOverrides() (-want +got):\n%s", diff)
		}
		other, err := f.Store.ListPartOverrides(ctx, f.Other, g)
		if err != nil || len(other) != 0 {
			t.Fatalf("other user's overrides = %v, %v, want none", other, err)
		}

		if err := f.Store.DeletePartOverride(ctx, f.UserID, g, "voice.md"); err != nil {
			t.Fatal(err)
		}
		got, err = f.Store.ListPartOverrides(ctx, f.UserID, g)
		if err != nil || len(got) != 0 {
			t.Fatalf("overrides after delete = %v, %v, want none", got, err)
		}
	})
}
