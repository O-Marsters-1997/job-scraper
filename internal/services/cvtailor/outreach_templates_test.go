package cvtailor_test

import (
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func TestService_CreateOutreachTemplate(t *testing.T) {
	t.Parallel()
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil, nil)
	valid := dto.OutreachTemplateInput{Kind: dto.OutreachReferral, Name: " Ask ", Body: "Hi {first_name}"}

	t.Run("trims and creates the kind's first template as the default", func(t *testing.T) {
		t.Parallel()
		got, err := svc.CreateOutreachTemplate(t.Context(), userID, valid)
		if err != nil {
			t.Fatalf("CreateOutreachTemplate() error = %v", err)
		}
		if got.Name != "Ask" || !got.IsDefault {
			t.Errorf("CreateOutreachTemplate() = %+v, want name Ask and default", got)
		}
	})

	for name, in := range map[string]dto.OutreachTemplateInput{
		"unknown kind": {Kind: "other", Name: "n", Body: "b"},
		"blank name":   {Kind: dto.OutreachReferral, Name: "  ", Body: "b"},
		"long name":    {Kind: dto.OutreachReferral, Name: strings.Repeat("n", 101), Body: "b"},
		"blank body":   {Kind: dto.OutreachReferral, Name: "n", Body: " "},
		"long body":    {Kind: dto.OutreachReferral, Name: "n", Body: strings.Repeat("b", 2001)},
	} {
		t.Run(name+" is invalid", func(t *testing.T) {
			t.Parallel()
			_, err := svc.CreateOutreachTemplate(t.Context(), userID, in)
			if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
				t.Errorf("CreateOutreachTemplate() error = %v, want invalid", err)
			}
		})
	}
}

func TestService_DeleteOutreachTemplate(t *testing.T) {
	t.Parallel()
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil, nil)
	ctx := t.Context()
	first, err := svc.CreateOutreachTemplate(ctx, userID, dto.OutreachTemplateInput{Kind: dto.OutreachFollowUp, Name: "One", Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateOutreachTemplate(ctx, userID, dto.OutreachTemplateInput{Kind: dto.OutreachFollowUp, Name: "Two", Body: "b"})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DeleteOutreachTemplate(ctx, userID, first.ID)
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindConflict.Status() {
		t.Errorf("DeleteOutreachTemplate(default) error = %v, want conflict", err)
	}
	if err := svc.DeleteOutreachTemplate(ctx, userID, second.ID); err != nil {
		t.Errorf("DeleteOutreachTemplate(non-default) error = %v, want nil", err)
	}
	err = svc.DeleteOutreachTemplate(ctx, userID, second.ID)
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
		t.Errorf("DeleteOutreachTemplate(gone) error = %v, want not found", err)
	}
}

func TestModule_SeedDefaults(t *testing.T) {
	t.Parallel()
	st := cvtailortest.NewFakeStore()

	if err := cvtailor.Build(cvtailor.Deps{Store: st}).SeedDefaults(t.Context(), nil, userID); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	got, err := st.ListOutreachTemplates(t.Context(), userID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[dto.OutreachKind]int{}
	for _, tpl := range got {
		if tpl.IsDefault {
			kinds[tpl.Kind]++
		}
	}
	if len(got) != 4 || len(kinds) != 4 {
		t.Errorf("seeded %d templates with defaults per kind %v, want 4 and one default each", len(got), kinds)
	}
}
