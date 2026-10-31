package cvtailor_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

type cachedAnswers map[string]dto.Answer

func (c cachedAnswers) CachedAnswers(context.Context, string, []string) (map[string]dto.Answer, error) {
	return c, nil
}

type neverAsks struct{ t *testing.T }

func (a neverAsks) Ask(context.Context, string, string, []string) (map[string]dto.Answer, error) {
	a.t.Error("drafting called the Answerer")
	return nil, nil
}

func TestModule_OutreachDraft(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	type env struct {
		mod     *cvtailor.Module
		store   *cvtailortest.FakeStore
		first   dto.Achievement
		newest  dto.Achievement
		templID map[dto.OutreachKind]string
	}
	newEnv := func(t *testing.T, cached cvtailor.CachedAsker) env {
		t.Helper()
		st := cvtailortest.NewFakeStore()
		pos, err := st.CreatePosition(ctx, userID, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
		if err != nil {
			t.Fatal(err)
		}
		first, err := st.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: pos.ID, Text: "cut p99 latency by 40%"})
		if err != nil {
			t.Fatal(err)
		}
		newest, err := st.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: pos.ID, Text: "shipped a billing service"})
		if err != nil {
			t.Fatal(err)
		}
		if err := st.SeedOutreachTemplates(ctx, nil, userID); err != nil {
			t.Fatal(err)
		}
		tpls, err := st.ListOutreachTemplates(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		ids := map[dto.OutreachKind]string{}
		for _, tpl := range tpls {
			ids[tpl.Kind] = tpl.ID
		}
		return env{
			mod:   cvtailor.Build(cvtailor.Deps{Store: st, Asker: neverAsks{t}, Cached: cached}),
			store: st, first: first, newest: newest, templID: ids,
		}
	}

	t.Run("a hiring manager draft fills the merge fields with the newest Achievement", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, cachedAnswers{})
		id := e.store.SeedOutreachSubject(userID, dto.OutreachSubject{Name: "Sam Jones", Role: "CTO", Company: "Globex"})

		got, err := e.mod.OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: id})
		if err != nil {
			t.Fatalf("OutreachDraft() error = %v", err)
		}

		want := dto.OutreachDraft{
			TemplateID: e.templID[dto.OutreachHiringManager], Kind: dto.OutreachHiringManager, AchievementID: e.newest.ID, HasHookSlot: true, HookFindings: []dto.LetterFinding{},
			Body: "Hi Sam, [hook] I shipped a billing service, and I'd love to bring that to Globex. Worth 15 minutes?",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("OutreachDraft() (-want +got):\n%s", diff)
		}
	})

	t.Run("the top cached lean wins and the Answerer is never called", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, cachedAnswers{
			"Would this job value a candidate who: cut p99 latency by 40%":    {PYes: 0.7, PNo: 0.1},
			"Would this job value a candidate who: shipped a billing service": {PYes: 0.2, PNo: 0.3},
		})
		id := e.store.SeedOutreachSubject(userID, dto.OutreachSubject{Name: "Pat", Role: "Talent Partner", Company: "Initech", JobID: "job-1", JobTitle: "Backend Engineer"})

		got, err := e.mod.OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: id})
		if err != nil {
			t.Fatalf("OutreachDraft() error = %v", err)
		}

		want := "Hi Pat, I'm a backend engineer looking at Initech's Backend Engineer roles. Recently I cut p99 latency by 40%. Open to a quick chat about what you're hiring for?"
		if got.Kind != dto.OutreachRecruiter || got.AchievementID != e.first.ID || got.Body != want {
			t.Errorf("OutreachDraft() = %+v, want recruiter draft with the cached top-lean Achievement", got)
		}
	})

	t.Run("replies since the last send suggest the follow up kind", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, cachedAnswers{})
		id := e.store.SeedOutreachSubject(userID, dto.OutreachSubject{Name: "Sam", Role: "Recruiter", Company: "Globex", SentSinceReply: 1})

		got, err := e.mod.OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: id})
		if err != nil {
			t.Fatalf("OutreachDraft() error = %v", err)
		}

		if got.Kind != dto.OutreachFollowUp || got.HasHookSlot {
			t.Errorf("OutreachDraft() = %+v, want follow_up without a hook slot", got)
		}
	})

	t.Run("a chosen template and Achievement override the suggestion", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, cachedAnswers{})
		id := e.store.SeedOutreachSubject(userID, dto.OutreachSubject{Name: "Sam", Role: "CTO", Company: "Globex"})

		got, err := e.mod.OutreachDraft(ctx, userID, dto.DraftQuery{ContactID: id, TemplateID: e.templID[dto.OutreachRecruiter], AchievementID: e.first.ID})
		if err != nil {
			t.Fatalf("OutreachDraft() error = %v", err)
		}

		want := "Hi Sam, I'm a backend engineer looking at Globex's CTO roles. Recently I cut p99 latency by 40%. Open to a quick chat about what you're hiring for?"
		if got.Kind != dto.OutreachRecruiter || got.AchievementID != e.first.ID || got.Body != want {
			t.Errorf("OutreachDraft() = %+v, want the chosen recruiter template and Achievement", got)
		}
	})

	t.Run("an unknown contact or achievement is not found", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, cachedAnswers{})
		id := e.store.SeedOutreachSubject(userID, dto.OutreachSubject{Name: "Sam"})
		for name, q := range map[string]dto.DraftQuery{
			"contact":     {ContactID: "missing"},
			"achievement": {ContactID: id, AchievementID: "missing"},
		} {
			_, err := e.mod.OutreachDraft(ctx, userID, q)
			if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
				t.Errorf("%s: OutreachDraft() error = %v, want not found", name, err)
			}
		}
	})
}
