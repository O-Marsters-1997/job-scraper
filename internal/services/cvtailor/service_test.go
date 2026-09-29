package cvtailor_test

import (
	"context"
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
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil, nil)
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
			_, err := svc.CreatePosition(context.Background(), "u1", tc.in)
			if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
				t.Fatalf("CreatePosition() err = %v, want an invalid error", err)
			}
		})
	}
}

func TestCreatePositionTreatsBlankDatesAsCurrent(t *testing.T) {
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil, nil)
	got, err := svc.CreatePosition(context.Background(), "u1", dto.PositionInput{
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
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil, nil)
	_, err := svc.CreateAchievement(context.Background(), "u1", dto.AchievementInput{PositionID: "p", Text: " "})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("CreateAchievement() err = %v, want an invalid error", err)
	}
}

func TestReorderRejectsRepeatedIDs(t *testing.T) {
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil, nil)
	_, err := svc.ReorderPositions(context.Background(), "u1", dto.ReorderInput{IDs: []string{"a", "a"}})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("ReorderPositions() err = %v, want an invalid error", err)
	}
}

const (
	user    = handlerstest.UserID
	jobID   = "job-1"
	docID   = "doc-1"
	tabID   = "t.0"
	heading = "Engineer, Acme"
)

type draftEnv struct {
	store *cvtailortest.FakeStore
	drive *cvtailortest.Drive
	svc   *cvtailor.Service
	pos   dto.Position
	input dto.DraftInput
}

func newDraftEnv(t *testing.T) draftEnv {
	t.Helper()
	store := cvtailortest.NewFakeStore()
	ctx := context.Background()
	pos, err := store.CreatePosition(ctx, user, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Cut p99 latency by moving queries to Postgres", "Mentored four engineers"} {
		a, err := store.CreateAchievement(ctx, user, dto.AchievementInput{PositionID: pos.ID, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		pos.Achievements = append(pos.Achievements, a)
	}
	if err := store.SaveHeadingMappings(ctx, user, docID, tabID, []dto.HeadingMapping{{HeadingText: heading, PositionID: &pos.ID}}); err != nil {
		t.Fatal(err)
	}
	store.SetJob(jobID, "We need a Go engineer.", "fp-1")
	drive := newDrive()
	return draftEnv{
		store: store,
		drive: drive,
		svc:   cvtailor.NewService(store, nil, nil, drive),
		pos:   pos,
		input: dto.DraftInput{JobID: jobID, DocID: docID, TabID: tabID, AchievementIDs: []string{pos.Achievements[0].ID}},
	}
}

func TestCreateDraft(t *testing.T) {
	t.Run("queues a pending Draft for confirmed achievements", func(t *testing.T) {
		e := newDraftEnv(t)

		id := e.queue(t)

		want := dto.Draft{ID: id, JobID: jobID, Status: "pending", Findings: []dto.DraftFinding{}}
		if diff := cmp.Diff(want, e.draft(t, id), cmpopts.IgnoreFields(dto.Draft{}, "CreatedAt")); diff != "" {
			t.Errorf("GetDraft(%s) mismatch (-want +got):\n%s", id, diff)
		}
	})

	e := newDraftEnv(t)
	other, err := e.store.CreatePosition(context.Background(), user, dto.PositionInput{Employer: "Unmapped", Title: "Dev"})
	if err != nil {
		t.Fatal(err)
	}
	unmapped, err := e.store.CreateAchievement(context.Background(), user, dto.AchievementInput{PositionID: other.ID, Text: "Shipped"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		user   string
		mutate func(*dto.DraftInput)
	}{
		{"no achievements is invalid", user, func(in *dto.DraftInput) { in.AchievementIDs = nil }},
		{"repeated achievement is invalid", user, func(in *dto.DraftInput) { in.AchievementIDs = []string{in.AchievementIDs[0], in.AchievementIDs[0]} }},
		{"another user's achievement is invalid", "user-2", func(*dto.DraftInput) {}},
		{"achievement of an unmapped position is invalid", user, func(in *dto.DraftInput) { in.AchievementIDs = []string{unmapped.ID} }},
		{"missing tab is invalid", user, func(in *dto.DraftInput) { in.TabID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := e.input
			in.AchievementIDs = append([]string(nil), in.AchievementIDs...)
			tc.mutate(&in)

			_, err := e.svc.CreateDraft(context.Background(), tc.user, in)

			if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
				t.Errorf("CreateDraft(%+v) error = %v, want an invalid error", in, err)
			}
		})
	}
}

func TestGetDraft(t *testing.T) {
	t.Run("links the Doc of a ready Draft", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		claim, err := e.store.ClaimDraft(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := e.store.CompleteDraft(context.Background(), claim, dto.DraftResult{DraftDocID: "copy-1"}); err != nil {
			t.Fatal(err)
		}

		got := e.draft(t, id)

		want := "https://docs.google.com/document/d/copy-1/edit"
		if got.DraftDocURL == nil || *got.DraftDocURL != want {
			t.Errorf("GetDraft(%s).DraftDocURL = %v, want %q", id, got.DraftDocURL, want)
		}
	})

	t.Run("hides the Doc of a Draft that is not ready", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		claim, err := e.store.ClaimDraft(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := e.store.SetDraftDoc(context.Background(), claim, "copy-1"); err != nil {
			t.Fatal(err)
		}

		got := e.draft(t, id)

		if got.DraftDocURL != nil {
			t.Errorf("GetDraft(%s).DraftDocURL = %q, want none before the Draft is ready", id, *got.DraftDocURL)
		}
	})

	t.Run("another user's Draft is not found", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)

		_, err := e.svc.GetDraft(context.Background(), "user-2", dto.DraftQuery{ID: id})

		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
			t.Errorf("GetDraft(%s) as another user error = %v, want a not-found error", id, err)
		}
	})
}
