package cvtailortest

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func runVariantContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	wantNotFound := func(t *testing.T, call string, err error) {
		t.Helper()
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("%s err = %v, want not found", call, err)
		}
	}
	seed := func(t *testing.T, f Fixture, userID string) dto.Achievement {
		t.Helper()
		p, err := f.Store.CreatePosition(ctx, userID, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
		if err != nil {
			t.Fatal(err)
		}
		a, err := f.Store.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: p.ID, Text: "default"})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	listed := func(t *testing.T, f Fixture) []dto.AchievementVariant {
		t.Helper()
		got, err := f.Store.ListPositions(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		return got[0].Achievements[0].Variants
	}

	t.Run("variants are listed oldest first and can be edited and deleted", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		a := seed(t, f, f.UserID)
		if got := listed(t, f); got == nil || len(got) != 0 {
			t.Fatalf("variants = %#v, want empty non-nil", got)
		}
		first, err := f.Store.CreateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{AchievementID: a.ID, Text: "one", Label: "short"})
		if err != nil {
			t.Fatal(err)
		}
		second, err := f.Store.CreateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{AchievementID: a.ID, Text: "two"})
		if err != nil {
			t.Fatal(err)
		}
		edited, err := f.Store.UpdateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{ID: first.ID, Text: "uno", Label: "long"})
		if err != nil {
			t.Fatal(err)
		}
		want := []dto.AchievementVariant{
			{ID: first.ID, AchievementID: a.ID, Text: "uno", Label: "long"},
			{ID: second.ID, AchievementID: a.ID, Text: "two"},
		}
		if diff := cmp.Diff(want, listed(t, f)); diff != "" {
			t.Fatalf("variants (-want +got):\n%s", diff)
		}
		if edited != want[0] {
			t.Fatalf("UpdateAchievementVariant() = %+v, want %+v", edited, want[0])
		}
		if err := f.Store.DeleteAchievementVariant(ctx, f.UserID, first.ID); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want[1:], listed(t, f)); diff != "" {
			t.Fatalf("variants after delete (-want +got):\n%s", diff)
		}
	})

	t.Run("deleting an achievement removes its variants", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		a := seed(t, f, f.UserID)
		v, err := f.Store.CreateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{AchievementID: a.ID, Text: "one"})
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Store.DeleteAchievement(ctx, f.UserID, a.ID); err != nil {
			t.Fatal(err)
		}
		wantNotFound(t, "DeleteAchievementVariant(cascaded)", f.Store.DeleteAchievementVariant(ctx, f.UserID, v.ID))
	})

	t.Run("missing and foreign ids return not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		theirs := seed(t, f, f.Other)
		v, err := f.Store.CreateAchievementVariant(ctx, f.Other, dto.AchievementVariantInput{AchievementID: theirs.ID, Text: "theirs"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Store.CreateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{AchievementID: missingID, Text: "x"})
		wantNotFound(t, "CreateAchievementVariant(missing)", err)
		_, err = f.Store.CreateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{AchievementID: theirs.ID, Text: "x"})
		wantNotFound(t, "CreateAchievementVariant(foreign)", err)
		_, err = f.Store.UpdateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{ID: missingID, Text: "x"})
		wantNotFound(t, "UpdateAchievementVariant(missing)", err)
		_, err = f.Store.UpdateAchievementVariant(ctx, f.UserID, dto.AchievementVariantInput{ID: v.ID, Text: "x"})
		wantNotFound(t, "UpdateAchievementVariant(foreign)", err)
		wantNotFound(t, "DeleteAchievementVariant(missing)", f.Store.DeleteAchievementVariant(ctx, f.UserID, missingID))
		wantNotFound(t, "DeleteAchievementVariant(foreign)", f.Store.DeleteAchievementVariant(ctx, f.UserID, v.ID))
	})
}
