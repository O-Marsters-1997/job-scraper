package cvtailor_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

const widgetsHeading = "Analyst, Widgets"

type rolesEnv struct {
	draftEnv
	widgets dto.Position
	tab     json.RawMessage
	id      string
}

func newRolesEnv(t *testing.T) rolesEnv {
	t.Helper()
	e := newDraftEnv(t)
	widgets, err := e.store.CreatePosition(t.Context(), userID, dto.PositionInput{Employer: "Widgets", Title: "Analyst"})
	if err != nil {
		t.Fatal(err)
	}
	a, err := e.store.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: widgets.ID, Text: "Built reports for finance"})
	if err != nil {
		t.Fatal(err)
	}
	widgets.Achievements = append(widgets.Achievements, a)
	mappings := []dto.HeadingMapping{{HeadingText: heading, PositionID: &e.pos.ID}, {HeadingText: widgetsHeading, PositionID: &widgets.ID}}
	if err := e.store.SaveHeadingMappings(t.Context(), userID, docID, tabID, mappings); err != nil {
		t.Fatal(err)
	}
	tab := baseTab(t, head(widgetsHeading), bullet("Built reports"), prose("Reference available"))
	e.input.AchievementIDs = []string{e.pos.Achievements[0].ID, a.ID}
	id := e.queue(t)
	result := cvedit.Result{Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{
		{PositionID: e.pos.ID, Bullets: []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: "Cut p99 latency by moving queries to Postgres"}}},
		{PositionID: widgets.ID, Bullets: []cvedit.Bullet{{AchievementIDs: []string{a.ID}, Text: "Built reports for finance"}}},
	}}}
	e.run(t, tick{docs: cvtailortest.Docs{TabJSON: tab}, editor: cvtailortest.Editing(result)})
	return rolesEnv{draftEnv: e, widgets: widgets, tab: tab, id: id}
}

func (e rolesEnv) save(t *testing.T, roles []string) (dto.Draft, error) {
	t.Helper()
	svc := cvtailor.NewService(e.store, cvtailortest.Docs{TabJSON: e.tab}, nil, e.drive)
	return svc.SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: e.id, Roles: roles})
}

func rebuiltBody(t *testing.T, e rolesEnv) string {
	t.Helper()
	for _, batch := range e.drive.Updates {
		for _, raw := range batch {
			var req map[string]map[string]any
			if err := json.Unmarshal(raw, &req); err != nil {
				t.Fatal(err)
			}
			if text, ok := req["insertText"]["text"].(string); ok && strings.Contains(text, "\n") {
				return text
			}
		}
	}
	return ""
}

func positionIDs(d dto.Draft) []string {
	var ids []string
	for _, p := range d.Content.Positions {
		ids = append(ids, p.PositionID)
	}
	return ids
}

func rebuilds(t *testing.T, e rolesEnv) int {
	t.Helper()
	n := 0
	for _, batch := range e.drive.Updates {
		for _, raw := range batch {
			var req map[string]json.RawMessage
			if err := json.Unmarshal(raw, &req); err != nil {
				t.Fatal(err)
			}
			if _, ok := req["deleteParagraphBullets"]; ok {
				n++
			}
		}
	}
	return n
}

func TestSaveDraftSlotsRoles(t *testing.T) {
	t.Parallel()
	t.Run("reordering moves each role block whole and keeps the bullet edits", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)

		got, err := e.save(t, []string{e.widgets.ID, e.pos.ID})
		if err != nil {
			t.Fatalf("SaveDraftSlots() error = %v", err)
		}

		wantBody := strings.Join([]string{widgetsHeading, "Built reports", "Reference available", heading,
			"Built and maintained the public APIs for the platform", "Ran on-call", "Wrote docs"}, "\n")
		if body := rebuiltBody(t, e); body != wantBody {
			t.Errorf("rebuilt body = %q, want %q", body, wantBody)
		}
		if inserted := insertedText(t, e.drive); !slices.Contains(inserted, "Cut p99 latency by moving queries to Postgres") {
			t.Errorf("reapplied edits inserted %q, want the generated bullet to survive the reorder", inserted)
		}
		if diff := cmp.Diff([]string{e.widgets.ID, e.pos.ID}, positionIDs(got)); diff != "" {
			t.Errorf("SaveDraftSlots().Content positions (-want +got):\n%s", diff)
		}
	})

	t.Run("a role left out is removed from the Doc, content and provenance", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)

		got, err := e.save(t, []string{e.pos.ID})
		if err != nil {
			t.Fatalf("SaveDraftSlots() error = %v", err)
		}

		if body := rebuiltBody(t, e); strings.Contains(body, "Widgets") || strings.Contains(body, "Built reports") {
			t.Errorf("rebuilt body = %q, want the Widgets block gone", body)
		}
		if diff := cmp.Diff([]string{e.pos.ID}, positionIDs(got)); diff != "" {
			t.Errorf("SaveDraftSlots().Content positions (-want +got):\n%s", diff)
		}
		if n := len(got.Provenance.Positions); n != 1 || got.Provenance.Positions[0].PositionID != e.pos.ID {
			t.Errorf("SaveDraftSlots().Provenance positions = %+v, want only the kept role", got.Provenance.Positions)
		}
	})

	t.Run("saving the order the Doc already has does not rebuild it", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)

		if _, err := e.save(t, []string{e.pos.ID, e.widgets.ID}); err != nil {
			t.Fatalf("SaveDraftSlots() error = %v", err)
		}

		if n := rebuilds(t, e); n != 0 {
			t.Errorf("rebuilds after saving the existing order = %d, want 0", n)
		}
	})

	t.Run("a save after a role was removed still works", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)
		if _, err := e.save(t, []string{e.pos.ID}); err != nil {
			t.Fatalf("SaveDraftSlots(roles) error = %v", err)
		}

		if _, err := e.save(t, nil); err != nil {
			t.Errorf("SaveDraftSlots() after removal error = %v", err)
		}
	})

	t.Run("adding a bank role clones a heading and writes its bullet after the last role", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)
		initech, err := e.store.CreatePosition(t.Context(), userID, dto.PositionInput{Employer: "Initech", Title: "Consultant"})
		if err != nil {
			t.Fatal(err)
		}
		a, err := e.store.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: initech.ID, Text: "Cut report time by half"})
		if err != nil {
			t.Fatal(err)
		}
		roles := []string{e.pos.ID, e.widgets.ID, initech.ID}
		svc := cvtailor.NewService(e.store, cvtailortest.Docs{TabJSON: e.tab}, nil, e.drive)
		add := dto.DraftSlotsInput{ID: e.id, Roles: roles, Positions: []dto.DraftPosition{{
			PositionID: initech.ID, Bullets: []dto.DraftBullet{{Text: "Cut report time by half", AchievementIDs: []string{a.ID}}, {Text: "Ran the audit", AchievementIDs: []string{a.ID}}},
		}}}

		got, err := svc.SaveDraftSlots(t.Context(), userID, add)
		if err != nil {
			t.Fatalf("SaveDraftSlots() error = %v", err)
		}

		wantBody := strings.Join([]string{heading, "Built and maintained the public APIs for the platform", "Ran on-call", "Wrote docs",
			widgetsHeading, "Built reports", "Reference available", "Consultant, Initech", "Built and maintained the public APIs for the platform"}, "\n")
		if body := rebuiltBody(t, e); body != wantBody {
			t.Errorf("rebuilt body = %q, want %q", body, wantBody)
		}
		if inserted := insertedText(t, e.drive); !slices.Contains(inserted, "Cut report time by half") {
			t.Errorf("inserted %q, want the new role's first bullet", inserted)
		}
		if diff := cmp.Diff(roles, positionIDs(got)); diff != "" {
			t.Errorf("SaveDraftSlots().Content positions (-want +got):\n%s", diff)
		}

		again, err := svc.SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: e.id, Roles: roles})
		if err != nil {
			t.Fatalf("SaveDraftSlots() with the role already added error = %v", err)
		}
		if diff := cmp.Diff(roles, positionIDs(again)); diff != "" {
			t.Errorf("SaveDraftSlots() positions after reload (-want +got):\n%s", diff)
		}
	})

	t.Run("a role added without a bullet is invalid", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)
		initech, err := e.store.CreatePosition(t.Context(), userID, dto.PositionInput{Employer: "Initech", Title: "Consultant"})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := e.save(t, []string{e.pos.ID, e.widgets.ID, initech.ID}); !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("SaveDraftSlots() error = %v, want an invalid error", err)
		}
	})

	t.Run("an empty, repeated or unknown role is invalid", func(t *testing.T) {
		t.Parallel()
		e := newRolesEnv(t)
		cases := map[string][]string{
			"empty":    {},
			"repeated": {e.pos.ID, e.pos.ID},
			"unknown":  {e.pos.ID, "nope"},
		}
		for name, roles := range cases {
			if _, err := e.save(t, roles); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("SaveDraftSlots(%s roles) error = %v, want an invalid error", name, err)
			}
		}
	})
}
