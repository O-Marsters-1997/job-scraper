package cvtailor_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func addAchievement(t *testing.T, st *cvtailortest.FakeStore, positionID, text string) dto.Achievement {
	t.Helper()
	a, err := st.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: positionID, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestGrillPrompt(t *testing.T) {
	t.Parallel()
	t.Run("holds the wrapper and the experience json", func(t *testing.T) {
		t.Parallel()
		svc, st := newService(t, nil, nil)
		a := addAchievement(t, st, addPosition(t, st, "Acme", "Engineer").ID, "Cut latency")
		got, err := svc.GrillPrompt(t.Context(), userID, dto.GrillPromptQuery{})
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"`grilling` skill", "Acme", a.ID, "Cut latency"} {
			if !strings.Contains(got.Text, want) {
				t.Errorf("GrillPrompt() lacks %q:\n%s", want, got.Text)
			}
		}
		if strings.Contains(got.Text, "## My CV") {
			t.Errorf("GrillPrompt() has a CV section with no Tab named")
		}
	})

	t.Run("adds the named CV Tab text", func(t *testing.T) {
		t.Parallel()
		docs := cvtailortest.Docs{TabJSON: tabJSON(t, head("Experience"), bullet("Shipped the billing rewrite"))}
		svc, _ := newService(t, docs, nil)
		got, err := svc.GrillPrompt(t.Context(), userID, dto.GrillPromptQuery{DocID: "d", TabID: "t"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got.Text, "## My CV\n\nExperience\nShipped the billing rewrite") {
			t.Errorf("GrillPrompt() lacks the CV text:\n%s", got.Text)
		}
	})

	t.Run("a doc id without a tab id is invalid", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService(t, nil, nil)
		_, err := svc.GrillPrompt(t.Context(), userID, dto.GrillPromptQuery{DocID: "d"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Fatalf("GrillPrompt() err = %v, want an invalid error", err)
		}
	})
}

func TestPreviewContext(t *testing.T) {
	t.Parallel()
	setup := func(t *testing.T) (*cvtailor.Service, dto.Achievement) {
		t.Helper()
		svc, st := newService(t, nil, nil)
		a := addAchievement(t, st, addPosition(t, st, "Acme", "Engineer").ID, "Cut latency")
		if err := st.SetAchievementContexts(t.Context(), userID, []string{a.ID}, []string{"old"}); err != nil {
			t.Fatal(err)
		}
		return svc, a
	}
	output := func(blocks ...string) string {
		var sb strings.Builder
		sb.WriteString("## notes\nprose\n")
		for _, b := range blocks {
			sb.WriteString("\n```json\n" + b + "\n```\n")
		}
		return sb.String()
	}

	t.Run("reads the last json block and returns before and after", func(t *testing.T) {
		t.Parallel()
		svc, a := setup(t)
		in := dto.ContextImportInput{Output: output(`{"achievements":[]}`, `{"achievements":[{"id":"`+a.ID+`","context":" new "}]}`)}
		got, err := svc.PreviewContext(t.Context(), userID, in)
		if err != nil {
			t.Fatal(err)
		}
		want := []dto.ContextChange{{AchievementID: a.ID, Text: "Cut latency", Before: "old", After: "new"}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("PreviewContext() (-want +got):\n%s", diff)
		}
	})

	t.Run("an id the user does not own fails the whole preview", func(t *testing.T) {
		t.Parallel()
		svc, a := setup(t)
		in := dto.ContextImportInput{Output: output(`{"achievements":[{"id":"` + a.ID + `","context":"x"},{"id":"nope","context":"y"}]}`)}
		_, err := svc.PreviewContext(t.Context(), userID, in)
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("PreviewContext() err = %v, want a not found error", err)
		}
	})

	t.Run("output without a valid json block is invalid", func(t *testing.T) {
		t.Parallel()
		svc, _ := setup(t)
		for _, out := range []string{"just prose", output(`{not json`)} {
			_, err := svc.PreviewContext(t.Context(), userID, dto.ContextImportInput{Output: out})
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Fatalf("PreviewContext(%q) err = %v, want an invalid error", out, err)
			}
		}
	})
}

func TestApplyContext(t *testing.T) {
	t.Parallel()
	t.Run("sets context and leaves text", func(t *testing.T) {
		t.Parallel()
		svc, st := newService(t, nil, nil)
		p := addPosition(t, st, "Acme", "Engineer")
		a := addAchievement(t, st, p.ID, "Cut latency")
		_, err := svc.ApplyContext(t.Context(), userID, dto.ContextApplyInput{
			Changes: []dto.ContextChange{{AchievementID: a.ID, Text: "ignored", After: " why "}},
		})
		if err != nil {
			t.Fatal(err)
		}
		got, err := st.ListPositions(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		want := dto.Achievement{ID: a.ID, PositionID: p.ID, Text: "Cut latency", Context: "why"}
		if diff := cmp.Diff(want, got[0].Achievements[0]); diff != "" {
			t.Fatalf("achievement (-want +got):\n%s", diff)
		}
	})

	t.Run("an unowned or repeated id fails", func(t *testing.T) {
		t.Parallel()
		svc, st := newService(t, nil, nil)
		a := addAchievement(t, st, addPosition(t, st, "Acme", "Engineer").ID, "Cut latency")
		_, err := svc.ApplyContext(t.Context(), userID, dto.ContextApplyInput{Changes: []dto.ContextChange{{AchievementID: "nope", After: "x"}}})
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("ApplyContext(unowned) err = %v, want a not found error", err)
		}
		_, err = svc.ApplyContext(t.Context(), userID, dto.ContextApplyInput{Changes: []dto.ContextChange{{AchievementID: a.ID}, {AchievementID: a.ID}}})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Fatalf("ApplyContext(repeat) err = %v, want an invalid error", err)
		}
	})
}
