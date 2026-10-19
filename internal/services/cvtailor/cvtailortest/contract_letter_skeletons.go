package cvtailortest

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func runLetterSkeletonContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	t.Run("an unsaved user gets an empty skeleton", func(t *testing.T) {
		f := newStore(t)
		got, err := f.Store.GetLetterSkeleton(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(dto.LetterSkeleton{Sections: []string{}}, got); diff != "" {
			t.Fatalf("GetLetterSkeleton() (-want +got):\n%s", diff)
		}
	})

	t.Run("save upserts and is scoped to the user", func(t *testing.T) {
		f := newStore(t)
		first := dto.LetterSkeleton{Intro: "Dear ___,", Conclusion: "Thanks, ___", Sections: []string{"fit", "why"}}
		if _, err := f.Store.SaveLetterSkeleton(ctx, f.UserID, first); err != nil {
			t.Fatal(err)
		}
		second := dto.LetterSkeleton{Intro: "Hi ___,", Sections: []string{"only"}}
		saved, err := f.Store.SaveLetterSkeleton(ctx, f.UserID, second)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(second, saved); diff != "" {
			t.Fatalf("SaveLetterSkeleton() (-want +got):\n%s", diff)
		}
		got, err := f.Store.GetLetterSkeleton(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(second, got); diff != "" {
			t.Fatalf("GetLetterSkeleton() after upsert (-want +got):\n%s", diff)
		}
		other, err := f.Store.GetLetterSkeleton(ctx, f.Other)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(dto.LetterSkeleton{Sections: []string{}}, other); diff != "" {
			t.Fatalf("other user's skeleton (-want +got):\n%s", diff)
		}
	})
}
