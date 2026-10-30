package cvtailortest

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

func runOutreachTemplateContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()
	byName := cmpopts.SortSlices(func(a, b dto.OutreachTemplate) bool { return a.Name < b.Name })
	ignoreID := cmpopts.IgnoreFields(dto.OutreachTemplate{}, "ID")

	wantErr := func(t *testing.T, call string, err, want error) {
		t.Helper()
		if !errors.Is(err, want) {
			t.Fatalf("%s err = %v, want %v", call, err, want)
		}
	}
	create := func(t *testing.T, f Fixture, kind dto.OutreachKind, name string) dto.OutreachTemplate {
		t.Helper()
		got, err := f.Store.CreateOutreachTemplate(ctx, f.UserID, dto.OutreachTemplateInput{Kind: kind, Name: name, Body: "Hi {first_name}"})
		if err != nil {
			t.Fatalf("CreateOutreachTemplate(%s): %v", name, err)
		}
		return got
	}

	t.Run("first of a kind is the default, later ones are not", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		first := create(t, f, dto.OutreachReferral, "Ask")
		second := create(t, f, dto.OutreachReferral, "Ask again")
		other := create(t, f, dto.OutreachFollowUp, "Bump")
		if !first.IsDefault || second.IsDefault || !other.IsDefault {
			t.Errorf("defaults = %v %v %v, want true false true", first.IsDefault, second.IsDefault, other.IsDefault)
		}
		_, err := f.Store.CreateOutreachTemplate(ctx, f.UserID, dto.OutreachTemplateInput{Kind: dto.OutreachReferral, Name: "Ask", Body: "x"})
		wantErr(t, "duplicate name", err, store.ErrTemplateNameTaken)
	})

	t.Run("update changes name and body, scoped to the user", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		tpl := create(t, f, dto.OutreachRecruiter, "Intro")
		got, err := f.Store.UpdateOutreachTemplate(ctx, f.UserID, dto.OutreachTemplateInput{ID: tpl.ID, Name: "Intro 2", Body: "New body"})
		if err != nil {
			t.Fatal(err)
		}
		want := dto.OutreachTemplate{ID: tpl.ID, Kind: dto.OutreachRecruiter, Name: "Intro 2", Body: "New body", IsDefault: true}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("UpdateOutreachTemplate() (-want +got):\n%s", diff)
		}
		_, err = f.Store.UpdateOutreachTemplate(ctx, f.Other, dto.OutreachTemplateInput{ID: tpl.ID, Name: "x", Body: "x"})
		wantErr(t, "other user", err, store.ErrTemplateNotFound)
		_, err = f.Store.UpdateOutreachTemplate(ctx, f.UserID, dto.OutreachTemplateInput{ID: missingID, Name: "x", Body: "x"})
		wantErr(t, "missing", err, store.ErrTemplateNotFound)
	})

	t.Run("make default moves the kind's default and nothing else", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		create(t, f, dto.OutreachRecruiter, "A")
		b := create(t, f, dto.OutreachRecruiter, "B")
		create(t, f, dto.OutreachFollowUp, "Bump")
		got, err := f.Store.MakeOutreachTemplateDefault(ctx, f.UserID, b.ID)
		if err != nil || !got.IsDefault {
			t.Fatalf("MakeOutreachTemplateDefault() = %+v, %v, want default", got, err)
		}
		list, err := f.Store.ListOutreachTemplates(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		want := []dto.OutreachTemplate{
			{Kind: dto.OutreachRecruiter, Name: "A", Body: "Hi {first_name}"},
			{Kind: dto.OutreachRecruiter, Name: "B", Body: "Hi {first_name}", IsDefault: true},
			{Kind: dto.OutreachFollowUp, Name: "Bump", Body: "Hi {first_name}", IsDefault: true},
		}
		if diff := cmp.Diff(want, list, byName, ignoreID); diff != "" {
			t.Errorf("ListOutreachTemplates() (-want +got):\n%s", diff)
		}
		_, err = f.Store.MakeOutreachTemplateDefault(ctx, f.Other, b.ID)
		wantErr(t, "other user", err, store.ErrTemplateNotFound)
	})

	t.Run("delete refuses a default while the kind has others", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		a := create(t, f, dto.OutreachRecruiter, "A")
		b := create(t, f, dto.OutreachRecruiter, "B")
		wantErr(t, "delete default", f.Store.DeleteOutreachTemplate(ctx, f.UserID, a.ID), store.ErrTemplateDefaultSet)
		wantErr(t, "other user", f.Store.DeleteOutreachTemplate(ctx, f.Other, b.ID), store.ErrTemplateNotFound)
		if err := f.Store.DeleteOutreachTemplate(ctx, f.UserID, b.ID); err != nil {
			t.Fatalf("delete non-default: %v", err)
		}
		if err := f.Store.DeleteOutreachTemplate(ctx, f.UserID, a.ID); err != nil {
			t.Fatalf("delete last of kind: %v", err)
		}
		wantErr(t, "already gone", f.Store.DeleteOutreachTemplate(ctx, f.UserID, a.ID), store.ErrTemplateNotFound)
	})
}
