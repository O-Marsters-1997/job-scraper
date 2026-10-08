package cvtailor_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func ptr(s string) *string { return &s }

func TestCreatePositionValidation(t *testing.T) {
	svc, _ := newService(t, nil, nil)
	cases := []struct {
		name string
		in   dto.PositionInput
	}{
		{"blank employer", dto.PositionInput{Employer: "  ", Title: "Engineer"}},
		{"blank title", dto.PositionInput{Employer: "Acme"}},
		{"bad date", dto.PositionInput{Employer: "Acme", Title: "Engineer", StartDate: ptr("last May")}},
		{"end before start", dto.PositionInput{Employer: "Acme", Title: "Engineer", StartDate: ptr("2022-01-01"), EndDate: ptr("2021-01-01")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreatePosition(t.Context(), userID, tc.in)
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Fatalf("CreatePosition() err = %v, want an invalid error", err)
			}
		})
	}
}

func TestCreatePositionTreatsBlankDatesAsCurrent(t *testing.T) {
	svc, _ := newService(t, nil, nil)
	got, err := svc.CreatePosition(t.Context(), userID, dto.PositionInput{
		Employer: " Acme ", Title: "Engineer", StartDate: ptr("2020-01-01"), EndDate: ptr(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Employer != "Acme" || got.EndDate != nil {
		t.Fatalf("CreatePosition() = %+v, want trimmed employer and nil end date", got)
	}
}

func TestCreateAchievementRejectsBlankText(t *testing.T) {
	svc, _ := newService(t, nil, nil)
	_, err := svc.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: "p", Text: " "})
	if !apperr.IsKind(err, apperr.KindInvalid) {
		t.Fatalf("CreateAchievement() err = %v, want an invalid error", err)
	}
}

func TestBankSkillRejectsBlankName(t *testing.T) {
	svc, _ := newService(t, nil, nil)
	_, err := svc.CreateBankSkill(t.Context(), userID, dto.BankSkillInput{Name: "  ", Category: "Languages"})
	if !apperr.IsKind(err, apperr.KindInvalid) {
		t.Fatalf("CreateBankSkill() err = %v, want an invalid error", err)
	}
}

func TestReorderRejectsRepeatedIDs(t *testing.T) {
	svc, _ := newService(t, nil, nil)
	_, err := svc.ReorderPositions(t.Context(), userID, dto.ReorderInput{IDs: []string{"a", "a"}})
	if !apperr.IsKind(err, apperr.KindInvalid) {
		t.Fatalf("ReorderPositions() err = %v, want an invalid error", err)
	}
}

const (
	userID      = handlerstest.UserID
	otherUserID = "user-2"
	jobID       = "job-1"
	docID       = "doc-1"
	tabID       = "t.0"
	heading     = "Engineer, Acme"
)

type draftEnv struct {
	store *cvtailortest.FakeStore
	drive *cvtailortest.Drive
	svc   *cvtailor.Service
	asker *fakeAsker
	pos   dto.Position
	input dto.DraftInput
}

func newDraftEnv(t *testing.T) draftEnv {
	t.Helper()
	store := cvtailortest.NewFakeStore()
	ctx := t.Context()
	pos, err := store.CreatePosition(ctx, userID, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Cut p99 latency by moving queries to Postgres", "Mentored four engineers"} {
		a, err := store.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: pos.ID, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		pos.Achievements = append(pos.Achievements, a)
	}
	if err := store.SaveHeadingMappings(ctx, userID, docID, tabID, []dto.HeadingMapping{{HeadingText: heading, PositionID: &pos.ID}}); err != nil {
		t.Fatal(err)
	}
	store.SetJob(dto.Job{ID: jobID, Title: "Platform Engineer", CompanySlug: "Acme", Description: "We need a Go engineer.", ContentFingerprint: "fp-1"})
	drive := newDrive()
	asker := &fakeAsker{}
	return draftEnv{
		store: store,
		drive: drive,
		svc:   cvtailor.NewService(store, cvtailortest.Docs{TabJSON: baseTab(t)}, asker, drive),
		asker: asker,
		pos:   pos,
		input: dto.DraftInput{JobID: jobID, DocID: docID, TabID: tabID, AchievementIDs: []string{pos.Achievements[0].ID}},
	}
}

func TestCreateDraft(t *testing.T) {
	t.Run("queues a pending Draft for confirmed achievements", func(t *testing.T) {
		e := newDraftEnv(t)

		id := e.queue(t)

		want := dto.Draft{ID: id, JobID: jobID, Status: "pending", Findings: []dto.DraftFinding{}}
		if diff := cmp.Diff(want, e.draft(t, id), cmpopts.IgnoreFields(dto.Draft{}, "CreatedAt", "BaseDocID", "BaseTabID", "AchievementIDs")); diff != "" {
			t.Errorf("GetDraft(%s) mismatch (-want +got):\n%s", id, diff)
		}
	})

	t.Run("records a label for every offered achievement", func(t *testing.T) {
		e := newDraftEnv(t)
		pos := e.pos
		extra := addAchievements(t, pos.ID, e.store, "Third", "Hand", "Unseen")
		e.asker.answers = map[string]dto.Answer{
			questionPrefix + pos.Achievements[0].Text: {PYes: 0.7, PNo: 0.1, PNotStated: 0.2, Confidence: 0.6},
			questionPrefix + pos.Achievements[1].Text: {PYes: 0.3, PNo: 0.1, PNotStated: 0.6, Confidence: 0.2},
			questionPrefix + "Third":                  {PYes: 0.2, PNo: 0.1, PNotStated: 0.7, Confidence: 0.1},
			questionPrefix + "Hand":                   {PYes: 0, PNo: 0.6, PNotStated: 0.4, Confidence: 0.5},
		}
		in := e.input
		in.AchievementIDs = []string{pos.Achievements[0].ID, extra[1].ID}

		if _, err := e.svc.CreateDraft(t.Context(), userID, in); err != nil {
			t.Fatalf("CreateDraft() error = %v", err)
		}

		answer := func(text string) *dto.Answer { a := e.asker.answers[questionPrefix+text]; return &a }
		want := []dto.BulletLabel{
			{AchievementID: pos.Achievements[0].ID, Answer: answer(pos.Achievements[0].Text), Preselected: true, Kept: true},
			{AchievementID: pos.Achievements[1].ID, Answer: answer(pos.Achievements[1].Text), Preselected: true},
			{AchievementID: extra[0].ID, Answer: answer("Third"), Preselected: true},
			{AchievementID: extra[2].ID},
			{AchievementID: extra[1].ID, Answer: answer("Hand"), Kept: true},
		}
		if diff := cmp.Diff(want, e.store.BulletLabels()); diff != "" {
			t.Errorf("recorded labels mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("still queues the Draft when Jev cannot answer", func(t *testing.T) {
		e := newDraftEnv(t)
		e.asker.err = errors.New("jev down")

		if _, err := e.svc.CreateDraft(t.Context(), userID, e.input); err != nil {
			t.Fatalf("CreateDraft() error = %v", err)
		}

		for _, l := range e.store.BulletLabels() {
			if l.Answer != nil {
				t.Errorf("label %+v has an answer, want none", l)
			}
		}
	})

	e := newDraftEnv(t)
	other, err := e.store.CreatePosition(t.Context(), userID, dto.PositionInput{Employer: "Unmapped", Title: "Dev"})
	if err != nil {
		t.Fatal(err)
	}
	unmapped, err := e.store.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: other.ID, Text: "Shipped"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		user   string
		mutate func(*dto.DraftInput)
	}{
		{"no achievements is invalid", userID, func(in *dto.DraftInput) { in.AchievementIDs = nil }},
		{"repeated achievement is invalid", userID, func(in *dto.DraftInput) { in.AchievementIDs = []string{in.AchievementIDs[0], in.AchievementIDs[0]} }},
		{"another user's achievement is invalid", otherUserID, func(*dto.DraftInput) {}},
		{"achievement of an unmapped position is invalid", userID, func(in *dto.DraftInput) { in.AchievementIDs = []string{unmapped.ID} }},
		{"missing tab is invalid", userID, func(in *dto.DraftInput) { in.TabID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := e.input
			in.AchievementIDs = append([]string(nil), in.AchievementIDs...)
			tc.mutate(&in)

			_, err := e.svc.CreateDraft(t.Context(), tc.user, in)

			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("CreateDraft(%+v) error = %v, want an invalid error", in, err)
			}
		})
	}
}

func TestGetDraft(t *testing.T) {
	t.Run("links the Doc of a ready Draft", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		claim, err := e.store.ClaimDraft(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if err := e.store.CompleteDraft(t.Context(), claim, dto.DraftResult{DraftDocID: "copy-1"}); err != nil {
			t.Fatal(err)
		}

		got := e.draft(t, id)

		want := "https://docs.google.com/document/d/copy-1/edit"
		if got.DraftDocURL == nil || *got.DraftDocURL != want {
			t.Errorf("GetDraft(%s).DraftDocURL = %v, want %q", id, got.DraftDocURL, want)
		}
	})

	t.Run("hides the Doc of a Draft that is not ready", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		claim, err := e.store.ClaimDraft(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if err := e.store.SetDraftDoc(t.Context(), claim, "copy-1"); err != nil {
			t.Fatal(err)
		}

		got := e.draft(t, id)

		if got.DraftDocURL != nil {
			t.Errorf("GetDraft(%s).DraftDocURL = %q, want none before the Draft is ready", id, *got.DraftDocURL)
		}
	})

	t.Run("another user's Draft is not found", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.GetDraft(t.Context(), otherUserID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("GetDraft(%s) as another user error = %v, want a not-found error", id, err)
		}
	})
}
