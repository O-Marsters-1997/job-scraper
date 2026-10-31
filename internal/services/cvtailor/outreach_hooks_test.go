package cvtailor_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/outreachhook"
)

const hookLine = "Acme's work on billing platforms is exactly the kind of problem I like, and the platform engineer role looks like the place to do more of it with a small team."

func TestModule_WriteHook(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	type env struct {
		store   *cvtailortest.FakeStore
		hooks   *cvtailortest.RecordingHooks
		contact string
		mod     func(creds cvtailor.Credentials) *cvtailor.Module
	}
	newEnv := func(t *testing.T) env {
		t.Helper()
		st := cvtailortest.NewFakeStore()
		pos, err := st.CreatePosition(ctx, userID, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: pos.ID, Text: "shipped a billing service"}); err != nil {
			t.Fatal(err)
		}
		if err := st.SeedOutreachTemplates(ctx, nil, userID); err != nil {
			t.Fatal(err)
		}
		st.SetJob(dto.Job{ID: "job-1", Title: "Platform Engineer", CompanySlug: "acme", Description: "<p>Own <b>billing</b> services.</p>"})
		hooks := &cvtailortest.RecordingHooks{Reply: outreachhook.Result{Text: hookLine, Cost: 0.004}}
		contact := st.SeedOutreachSubject(userID, dto.OutreachSubject{Name: "Sam Jones", Role: "CTO", Company: "Acme", JobID: "job-1", JobTitle: "Platform Engineer"})
		return env{store: st, hooks: hooks, contact: contact, mod: func(creds cvtailor.Credentials) *cvtailor.Module {
			return cvtailor.Build(cvtailor.Deps{Store: st, Asker: neverAsks{t}, Hooks: hooks, Creds: creds})
		}}
	}

	t.Run("the hook fills the draft, adds its cost and its prompt never names the contact", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		m := e.mod(apiKey)

		first, err := m.WriteHook(ctx, userID, dto.HookInput{ContactID: e.contact})
		if err != nil {
			t.Fatalf("WriteHook() error = %v", err)
		}
		again, err := m.WriteHook(ctx, userID, dto.HookInput{ContactID: e.contact})
		if err != nil {
			t.Fatalf("WriteHook() again error = %v", err)
		}

		if !strings.Contains(first.Body, "Hi Sam, "+hookLine+" I shipped a billing service") || first.Hook != hookLine {
			t.Errorf("WriteHook() = %+v, want the hook in the body", first)
		}
		if diff := cmp.Diff([]dto.LetterFinding{}, first.HookFindings); diff != "" {
			t.Errorf("HookFindings (-want +got):\n%s", diff)
		}
		if again.Cost != 0.008 {
			t.Errorf("Cost after two hooks = %v, want 0.008", again.Cost)
		}
		want := outreachhook.Input{
			JobTitle: "Platform Engineer", Company: "Acme", ContactRole: "CTO",
			JobDescription: "Own billing services.", Achievement: "shipped a billing service",
		}
		if diff := cmp.Diff(want, e.hooks.Inputs[0]); diff != "" {
			t.Errorf("hook input (-want +got):\n%s", diff)
		}
		if strings.Contains(fmt.Sprintf("%+v", e.hooks.Inputs[0]), "Sam") {
			t.Error("hook input carries the contact's name")
		}
	})

	t.Run("the stored hook shows on a later draft with its findings", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		e.hooks.Reply.Text = "Short and passionate."

		if _, err := e.mod(apiKey).WriteHook(ctx, userID, dto.HookInput{ContactID: e.contact}); err != nil {
			t.Fatalf("WriteHook() error = %v", err)
		}
		got, err := e.mod(apiKey).OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: e.contact})
		if err != nil {
			t.Fatalf("OutreachDraft() error = %v", err)
		}

		kinds := map[dto.LetterFindingKind]bool{}
		for _, f := range got.HookFindings {
			kinds[f.Kind] = true
		}
		if !strings.Contains(got.Body, "Short and passionate.") || !kinds[dto.FindingLength] || !kinds[dto.FindingAITell] || !kinds[dto.FindingMissingCompany] {
			t.Errorf("OutreachDraft() = %+v, want the stored hook with length, ai_tell and missing_company findings", got)
		}
	})

	t.Run("no key asks for one and the draft still renders the placeholder", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		m := e.mod(cvtailortest.NoKey{})

		_, err := m.WriteHook(ctx, userID, dto.HookInput{ContactID: e.contact})
		got, draftErr := m.OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: e.contact})

		if !apperr.IsKind(err, apperr.KindUnprocessable) || len(e.hooks.Inputs) != 0 {
			t.Errorf("WriteHook() = %v with %d model calls, want unprocessable and none", err, len(e.hooks.Inputs))
		}
		if draftErr != nil || !strings.Contains(got.Body, "[hook]") {
			t.Errorf("OutreachDraft() = %+v, %v, want the [hook] placeholder", got, draftErr)
		}
	})

	t.Run("a model failure is upstream and stores nothing", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		e.hooks.Err = data.ErrNotFound

		_, err := e.mod(apiKey).WriteHook(ctx, userID, dto.HookInput{ContactID: e.contact})
		got, _ := e.mod(apiKey).OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: e.contact})

		if !apperr.IsKind(err, apperr.KindUpstream) || got.Hook != "" {
			t.Errorf("WriteHook() = %v, hook %q, want upstream and no stored hook", err, got.Hook)
		}
	})

	t.Run("an empty reply is upstream", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		e.hooks.Reply.Text = ""

		_, err := e.mod(apiKey).WriteHook(ctx, userID, dto.HookInput{ContactID: e.contact})

		if !apperr.IsKind(err, apperr.KindUpstream) {
			t.Errorf("WriteHook() = %v, want upstream", err)
		}
	})

	t.Run("an unknown contact or achievement is not found", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		for name, in := range map[string]dto.HookInput{
			"contact":     {ContactID: "missing"},
			"achievement": {ContactID: e.contact, AchievementID: "missing"},
		} {
			_, err := e.mod(apiKey).WriteHook(ctx, userID, in)
			if !apperr.IsKind(err, apperr.KindNotFound) {
				t.Errorf("%s: WriteHook() error = %v, want not found", name, err)
			}
		}
	})
}
