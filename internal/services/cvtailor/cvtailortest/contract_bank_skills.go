package cvtailortest

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func runBankSkillContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	wantKind := func(t *testing.T, call string, err error, kind apperr.Kind) {
		t.Helper()
		if status, ok := apperr.StatusFor(err); !ok || status != kind.Status() {
			t.Fatalf("%s err = %v, want status %d", call, err, kind.Status())
		}
	}
	add := func(t *testing.T, f Fixture, userID, name, category string) dto.BankSkill {
		t.Helper()
		s, err := f.Store.CreateBankSkill(ctx, userID, dto.BankSkillInput{Name: name, Category: category})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	names := func(t *testing.T, f Fixture, userID string) []string {
		t.Helper()
		got, err := f.Store.ListBankSkills(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, s := range got {
			out = append(out, s.Name)
		}
		return out
	}

	t.Run("create appends and update renames and recategorises", func(t *testing.T) {
		f := newStore(t)
		add(t, f, f.UserID, "Go", "Languages")
		sk := add(t, f, f.UserID, "Postgress", "")
		got, err := f.Store.UpdateBankSkill(ctx, f.UserID, dto.BankSkillInput{ID: sk.ID, Name: "Postgres", Category: "Databases"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Postgres" || got.Category != "Databases" {
			t.Fatalf("UpdateBankSkill() = %+v, want Postgres / Databases", got)
		}
		if diff := cmp.Diff([]string{"Go", "Postgres"}, names(t, f, f.UserID)); diff != "" {
			t.Fatalf("names (-want +got):\n%s", diff)
		}
	})

	t.Run("a duplicate name ignoring case is a conflict", func(t *testing.T) {
		f := newStore(t)
		add(t, f, f.UserID, "Go", "")
		other := add(t, f, f.UserID, "Rust", "")
		_, err := f.Store.CreateBankSkill(ctx, f.UserID, dto.BankSkillInput{Name: "gO"})
		wantKind(t, "CreateBankSkill", err, apperr.KindConflict)
		_, err = f.Store.UpdateBankSkill(ctx, f.UserID, dto.BankSkillInput{ID: other.ID, Name: "GO"})
		wantKind(t, "UpdateBankSkill", err, apperr.KindConflict)
		if _, err := f.Store.UpdateBankSkill(ctx, f.UserID, dto.BankSkillInput{ID: other.ID, Name: "RUST", Category: "x"}); err != nil {
			t.Fatalf("UpdateBankSkill() recasing its own name err = %v, want nil", err)
		}
	})

	t.Run("reorder moves skills and rejects an incomplete list", func(t *testing.T) {
		f := newStore(t)
		a := add(t, f, f.UserID, "A", "")
		b := add(t, f, f.UserID, "B", "")
		c := add(t, f, f.UserID, "C", "")
		if err := f.Store.ReorderBankSkills(ctx, f.UserID, []string{c.ID, a.ID, b.ID}); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"C", "A", "B"}, names(t, f, f.UserID)); diff != "" {
			t.Fatalf("names (-want +got):\n%s", diff)
		}
		wantKind(t, "ReorderBankSkills", f.Store.ReorderBankSkills(ctx, f.UserID, []string{a.ID, b.ID}), apperr.KindInvalid)
	})

	t.Run("delete removes a skill", func(t *testing.T) {
		f := newStore(t)
		sk := add(t, f, f.UserID, "Go", "")
		if err := f.Store.DeleteBankSkill(ctx, f.UserID, sk.ID); err != nil {
			t.Fatal(err)
		}
		if got := names(t, f, f.UserID); len(got) != 0 {
			t.Fatalf("names after delete = %v, want empty", got)
		}
		wantKind(t, "DeleteBankSkill", f.Store.DeleteBankSkill(ctx, f.UserID, sk.ID), apperr.KindNotFound)
	})

	t.Run("another user's skills are invisible and untouchable", func(t *testing.T) {
		f := newStore(t)
		sk := add(t, f, f.UserID, "Go", "")
		if got := names(t, f, f.Other); len(got) != 0 {
			t.Fatalf("ListBankSkills(other) = %v, want empty", got)
		}
		_, err := f.Store.UpdateBankSkill(ctx, f.Other, dto.BankSkillInput{ID: sk.ID, Name: "Hijacked"})
		wantKind(t, "UpdateBankSkill(other)", err, apperr.KindNotFound)
		wantKind(t, "DeleteBankSkill(other)", f.Store.DeleteBankSkill(ctx, f.Other, sk.ID), apperr.KindNotFound)
		if _, err := f.Store.CreateBankSkill(ctx, f.Other, dto.BankSkillInput{Name: "Go"}); err != nil {
			t.Fatalf("CreateBankSkill(other) same name err = %v, want nil", err)
		}
		if diff := cmp.Diff([]string{"Go"}, names(t, f, f.UserID)); diff != "" {
			t.Fatalf("names (-want +got):\n%s", diff)
		}
	})
}
