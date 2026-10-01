package cvtailor_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

type countingDocs struct {
	cvtailor.DocFetcher
	gets int
}

func (c *countingDocs) GetDocument(ctx context.Context, userID, docID, tabID string) (json.RawMessage, error) {
	c.gets++
	return c.DocFetcher.GetDocument(ctx, userID, docID, tabID)
}

func slotIDsOf(l dto.DraftLayout) []string {
	out := make([]string, len(l.Blocks))
	for i, b := range l.Blocks {
		out[i] = b.SlotID
	}
	return out
}

// twoPositionDraft is a ready Draft whose first Position lost two of its
// three bullets in generation, so the Draft Doc's slots are numbered
// differently from the base CV's.
func twoPositionDraft(t *testing.T) (draftEnv, string, cvtailor.DocFetcher) {
	t.Helper()
	e := newDraftEnv(t)
	second, err := e.store.CreatePosition(t.Context(), userID, dto.PositionInput{Employer: "Beta", Title: "Dev"})
	if err != nil {
		t.Fatal(err)
	}
	ach, err := e.store.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: second.ID, Text: "Shipped billing"})
	if err != nil {
		t.Fatal(err)
	}
	mappings := []dto.HeadingMapping{{HeadingText: heading, PositionID: &e.pos.ID}, {HeadingText: "Beta", PositionID: &second.ID}}
	if err := e.store.SaveHeadingMappings(t.Context(), userID, docID, tabID, mappings); err != nil {
		t.Fatal(err)
	}
	e.input.AchievementIDs = []string{e.pos.Achievements[0].ID, ach.ID}
	base := tabJSON(t, head("Profile"), prose("Engineer who ships."), head(heading), bullet("a0"), bullet("a1"), bullet("a2"), head("Beta"), bullet("b0"), head("Skills"), prose("Go, SQL"))
	copyTab := tabJSON(t, head("Profile"), prose("Engineer who ships."), head(heading), bullet("Cut p99 latency by moving queries to Postgres"), head("Beta"), bullet("Shipped billing"), head("Skills"), prose("Go, SQL"))
	docs := cvtailortest.EditedCopyDocs{BaseDocID: docID, Base: base, Copy: copyTab}
	result := cvedit.Result{Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{
		{PositionID: e.pos.ID, Bullets: []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: "Cut p99 latency by moving queries to Postgres"}}},
		{PositionID: second.ID, Bullets: []cvedit.Bullet{{AchievementIDs: []string{ach.ID}, Text: "Shipped billing"}}},
	}}}
	id := e.queue(t)
	e.run(t, tick{docs: docs, editor: cvtailortest.Editing(result)})
	return e, id, docs
}

func TestDraftLayout(t *testing.T) {
	t.Run("maps Draft Doc slots to the base slot IDs the provenance uses", func(t *testing.T) {
		e, id, docs := twoPositionDraft(t)
		counted := &countingDocs{DocFetcher: docs}
		svc := cvtailor.NewService(e.store, counted, nil, e.drive)

		got, err := svc.DraftLayout(t.Context(), userID, dto.DraftQuery{ID: id})

		if err != nil {
			t.Fatalf("DraftLayout(%s) error = %v", id, err)
		}
		want := []string{"", "profile", "", "s0", "", "s3", "", ""}
		if diff := cmp.Diff(want, slotIDsOf(got)); diff != "" {
			t.Errorf("DraftLayout() slot IDs (-want +got):\n%s", diff)
		}
		if last := got.Blocks[len(got.Blocks)-1]; last.Section != "skills" || got.Blocks[2].Section != "" {
			t.Errorf("DraftLayout() sections = %q on the skills text and %q on a bullet, want skills only on the skills text", last.Section, got.Blocks[2].Section)
		}
		if counted.gets != 1 {
			t.Errorf("DraftLayout() fetched %d Docs, want 1", counted.gets)
		}
	})
}

func TestDraftLayoutRefusals(t *testing.T) {
	e, id, docs := twoPositionDraft(t)
	pending := e.queue(t)
	e2, discarded, _ := twoPositionDraft(t)
	if _, err := e2.svc.DiscardDraft(t.Context(), userID, dto.DraftQuery{ID: discarded}); err != nil {
		t.Fatal(err)
	}
	svc := cvtailor.NewService(e.store, docs, nil, e.drive)

	t.Run("a draft that is not ready or was discarded is a conflict", func(t *testing.T) {
		for name, c := range map[string]struct {
			svc *cvtailor.Service
			id  string
		}{"pending": {svc, pending}, "discarded": {cvtailor.NewService(e2.store, docs, nil, e2.drive), discarded}} {
			_, err := c.svc.DraftLayout(t.Context(), userID, dto.DraftQuery{ID: c.id})

			if !apperr.IsKind(err, apperr.KindConflict) {
				t.Errorf("DraftLayout(%s draft) error = %v, want a conflict", name, err)
			}
		}
	})

	t.Run("another user's draft is not found", func(t *testing.T) {
		_, err := svc.DraftLayout(t.Context(), otherUserID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("DraftLayout(%s) as another user error = %v, want a not-found error", id, err)
		}
	})

	t.Run("a Doc with a table is unprocessable and says why", func(t *testing.T) {
		table := cvtailortest.Docs{TabJSON: json.RawMessage(`{"documentTab":{"body":{"content":[{"startIndex":1,"endIndex":2,"table":{}}]}}}`)}

		_, err := cvtailor.NewService(e.store, table, nil, e.drive).DraftLayout(t.Context(), userID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindUnprocessable) || !strings.Contains(err.Error(), "non-paragraph") {
			t.Errorf("DraftLayout() error = %v, want an unprocessable error naming the reason", err)
		}
	})
}

func TestDraftLayoutRoute(t *testing.T) {
	e, id, docs := twoPositionDraft(t)
	r := newRouter(cvtailor.Deps{Store: e.store, Drive: e.drive, Docs: docs})

	got := handlerstest.Do[dto.DraftLayout](t, r, http.StatusOK, "GET /tailoring/drafts/"+id+"/layout", "")

	if len(got.Blocks) == 0 || got.Blocks[1].SlotID != "profile" {
		t.Errorf("GET layout = %+v, want the profile block tagged", got)
	}
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "GET /tailoring/drafts/missing/layout", "")
}
