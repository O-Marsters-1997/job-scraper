package cvtailortest

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
)

const missingID = "00000000-0000-0000-0000-00000000dead"

// Fixture is what RunStoreContract needs: a store and two users it can
// write for. The fake accepts any string; a real store needs rows that
// user_id's foreign key resolves to.
type Fixture struct {
	Store  cvtailor.Store
	UserID string
	Other  string
	// JobID is an existing Job; a real store needs a jobs row for it.
	JobID string
}

// RunStoreContract proves newStore's cvtailor.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	t.Run("positions", func(t *testing.T) { runPositionContract(t, newStore) })
	t.Run("bank skills", func(t *testing.T) { runBankSkillContract(t, newStore) })
	t.Run("heading mappings", func(t *testing.T) { runHeadingMappingContract(t, newStore) })
	t.Run("drafts", func(t *testing.T) { runDraftContract(t, newStore) })
}

func runPositionContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	wantKind := func(t *testing.T, call string, err error, kind apperr.Kind) {
		t.Helper()
		if status, ok := apperr.StatusFor(err); !ok || status != kind.Status() {
			t.Fatalf("%s err = %v, want status %d", call, err, kind.Status())
		}
	}
	wantNotFound := func(t *testing.T, call string, err error) {
		t.Helper()
		wantKind(t, call, err, apperr.KindNotFound)
	}
	position := func(t *testing.T, f Fixture, userID, employer string) dto.Position {
		t.Helper()
		p, err := f.Store.CreatePosition(ctx, userID, dto.PositionInput{Employer: employer, Title: "Engineer"})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	achievement := func(t *testing.T, f Fixture, positionID, text string) dto.Achievement {
		t.Helper()
		a, err := f.Store.CreateAchievement(ctx, f.UserID, dto.AchievementInput{PositionID: positionID, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	employers := func(t *testing.T, f Fixture) []string {
		t.Helper()
		got, err := f.Store.ListPositions(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, p := range got {
			out = append(out, p.Employer)
		}
		return out
	}
	texts := func(t *testing.T, f Fixture, positionID string) []string {
		t.Helper()
		got, err := f.Store.ListPositions(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, p := range got {
			if p.ID == positionID {
				for _, a := range p.Achievements {
					out = append(out, a.Text)
				}
			}
		}
		return out
	}

	t.Run("list is empty for a user with no positions", func(t *testing.T) {
		f := newStore(t)
		got, err := f.Store.ListPositions(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("ListPositions() = %+v, want empty", got)
		}
	})

	t.Run("create stores dates and puts the newest position first", func(t *testing.T) {
		f := newStore(t)
		start, end := "2020-01-01", "2022-06-30"
		first, err := f.Store.CreatePosition(ctx, f.UserID, dto.PositionInput{
			Employer: "Acme", Title: "Engineer", StartDate: &start, EndDate: &end,
		})
		if err != nil {
			t.Fatal(err)
		}
		if first.StartDate == nil || *first.StartDate != start || first.EndDate == nil || *first.EndDate != end {
			t.Fatalf("CreatePosition() dates = %v..%v, want %s..%s", first.StartDate, first.EndDate, start, end)
		}
		position(t, f, f.UserID, "Globex")
		if diff := cmp.Diff([]string{"Globex", "Acme"}, employers(t, f)); diff != "" {
			t.Fatalf("employers (-want +got):\n%s", diff)
		}
	})

	t.Run("update changes a position", func(t *testing.T) {
		f := newStore(t)
		p := position(t, f, f.UserID, "Acme")
		got, err := f.Store.UpdatePosition(ctx, f.UserID, dto.PositionInput{ID: p.ID, Employer: "Acme Ltd", Title: "Lead"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Employer != "Acme Ltd" || got.Title != "Lead" {
			t.Fatalf("UpdatePosition() = %+v, want Acme Ltd / Lead", got)
		}
	})

	t.Run("achievements append and reorder", func(t *testing.T) {
		f := newStore(t)
		p := position(t, f, f.UserID, "Acme")
		a := achievement(t, f, p.ID, "first")
		b := achievement(t, f, p.ID, "second")
		c := achievement(t, f, p.ID, "third")
		if diff := cmp.Diff([]string{"first", "second", "third"}, texts(t, f, p.ID)); diff != "" {
			t.Fatalf("texts (-want +got):\n%s", diff)
		}
		if err := f.Store.ReorderAchievements(ctx, f.UserID, p.ID, []string{c.ID, a.ID, b.ID}); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"third", "first", "second"}, texts(t, f, p.ID)); diff != "" {
			t.Fatalf("texts after reorder (-want +got):\n%s", diff)
		}
	})

	t.Run("import adds positions with achievements above the existing ones in order", func(t *testing.T) {
		f := newStore(t)
		position(t, f, f.UserID, "Existing")
		start := "2021-03-01"
		got, err := f.Store.ImportPositions(ctx, f.UserID, []dto.ImportPosition{
			{Employer: "Acme", Title: "Engineer", StartDate: &start, Achievements: []string{"one", "two"}},
			{Employer: "Globex", Title: "Intern"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Employer != "Acme" || got[1].Employer != "Globex" {
			t.Fatalf("ImportPositions() = %+v, want Acme then Globex", got)
		}
		if diff := cmp.Diff([]string{"Acme", "Globex", "Existing"}, employers(t, f)); diff != "" {
			t.Fatalf("employers (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"one", "two"}, texts(t, f, got[0].ID)); diff != "" {
			t.Fatalf("achievements (-want +got):\n%s", diff)
		}
	})

	t.Run("update achievement changes its text", func(t *testing.T) {
		f := newStore(t)
		p := position(t, f, f.UserID, "Acme")
		a := achievement(t, f, p.ID, "before")
		got, err := f.Store.UpdateAchievement(ctx, f.UserID, dto.AchievementInput{ID: a.ID, Text: "after"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Text != "after" {
			t.Fatalf("UpdateAchievement() text = %q, want after", got.Text)
		}
	})

	t.Run("reorder positions", func(t *testing.T) {
		f := newStore(t)
		a := position(t, f, f.UserID, "A")
		b := position(t, f, f.UserID, "B")
		c := position(t, f, f.UserID, "C")
		if err := f.Store.ReorderPositions(ctx, f.UserID, []string{a.ID, c.ID, b.ID}); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"A", "C", "B"}, employers(t, f)); diff != "" {
			t.Fatalf("employers (-want +got):\n%s", diff)
		}
	})

	t.Run("reorder rejects an incomplete or foreign id list", func(t *testing.T) {
		f := newStore(t)
		a := position(t, f, f.UserID, "A")
		position(t, f, f.UserID, "B")
		foreign := position(t, f, f.Other, "Theirs")
		wantKind(t, "ReorderPositions(partial)", f.Store.ReorderPositions(ctx, f.UserID, []string{a.ID}), apperr.KindInvalid)
		wantKind(t, "ReorderPositions(foreign)", f.Store.ReorderPositions(ctx, f.UserID, []string{a.ID, foreign.ID}), apperr.KindInvalid)
		wantKind(t, "ReorderPositions(duplicate)", f.Store.ReorderPositions(ctx, f.UserID, []string{a.ID, a.ID}), apperr.KindInvalid)
	})

	t.Run("delete position cascades its achievements", func(t *testing.T) {
		f := newStore(t)
		p := position(t, f, f.UserID, "Acme")
		a := achievement(t, f, p.ID, "gone")
		if err := f.Store.DeletePosition(ctx, f.UserID, p.ID); err != nil {
			t.Fatal(err)
		}
		if got := employers(t, f); len(got) != 0 {
			t.Fatalf("employers = %v, want none", got)
		}
		wantNotFound(t, "DeleteAchievement(cascaded)", f.Store.DeleteAchievement(ctx, f.UserID, a.ID))
	})

	t.Run("delete achievement removes it", func(t *testing.T) {
		f := newStore(t)
		p := position(t, f, f.UserID, "Acme")
		a := achievement(t, f, p.ID, "keep")
		b := achievement(t, f, p.ID, "drop")
		if err := f.Store.DeleteAchievement(ctx, f.UserID, b.ID); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{a.Text}, texts(t, f, p.ID)); diff != "" {
			t.Fatalf("texts (-want +got):\n%s", diff)
		}
	})

	t.Run("missing ids return not found", func(t *testing.T) {
		f := newStore(t)
		_, err := f.Store.UpdatePosition(ctx, f.UserID, dto.PositionInput{ID: missingID, Employer: "x", Title: "y"})
		wantNotFound(t, "UpdatePosition(missing)", err)
		wantNotFound(t, "DeletePosition(missing)", f.Store.DeletePosition(ctx, f.UserID, missingID))
		_, err = f.Store.CreateAchievement(ctx, f.UserID, dto.AchievementInput{PositionID: missingID, Text: "x"})
		wantNotFound(t, "CreateAchievement(missing position)", err)
		_, err = f.Store.UpdateAchievement(ctx, f.UserID, dto.AchievementInput{ID: missingID, Text: "x"})
		wantNotFound(t, "UpdateAchievement(missing)", err)
		wantNotFound(t, "DeleteAchievement(missing)", f.Store.DeleteAchievement(ctx, f.UserID, missingID))
		wantNotFound(t, "ReorderAchievements(missing position)", f.Store.ReorderAchievements(ctx, f.UserID, missingID, nil))
	})

	t.Run("another user's position and achievements return not found", func(t *testing.T) {
		f := newStore(t)
		p := position(t, f, f.Other, "Theirs")
		a, err := f.Store.CreateAchievement(ctx, f.Other, dto.AchievementInput{PositionID: p.ID, Text: "theirs"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = f.Store.UpdatePosition(ctx, f.UserID, dto.PositionInput{ID: p.ID, Employer: "x", Title: "y"})
		wantNotFound(t, "UpdatePosition(foreign)", err)
		wantNotFound(t, "DeletePosition(foreign)", f.Store.DeletePosition(ctx, f.UserID, p.ID))
		_, err = f.Store.CreateAchievement(ctx, f.UserID, dto.AchievementInput{PositionID: p.ID, Text: "x"})
		wantNotFound(t, "CreateAchievement(foreign position)", err)
		_, err = f.Store.UpdateAchievement(ctx, f.UserID, dto.AchievementInput{ID: a.ID, Text: "x"})
		wantNotFound(t, "UpdateAchievement(foreign)", err)
		wantNotFound(t, "DeleteAchievement(foreign)", f.Store.DeleteAchievement(ctx, f.UserID, a.ID))
		wantNotFound(t, "ReorderAchievements(foreign)", f.Store.ReorderAchievements(ctx, f.UserID, p.ID, []string{a.ID}))

		if got := employers(t, f); len(got) != 0 {
			t.Fatalf("ListPositions() leaked %v", got)
		}
	})
}
