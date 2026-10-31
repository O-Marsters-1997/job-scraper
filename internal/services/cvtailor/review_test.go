package cvtailor_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

func (e draftEnv) readyDrafts(t *testing.T, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		ids[i] = e.readyWith(t, tick{}, e.citedEdit("Cut p99 latency by quickly moving queries to Postgres"))
	}
	return ids
}

func TestGetDraftProvenance(t *testing.T) {
	t.Run("lists the cited achievements", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		d := e.draft(t, id)

		if d.Provenance == nil || len(d.Provenance.Positions) != 1 || len(d.Provenance.Positions[0].Bullets) != 1 {
			t.Fatalf("GetDraft().Provenance = %+v, want one position with one bullet", d.Provenance)
		}
		bullet := d.Provenance.Positions[0].Bullets[0]
		if len(bullet.Achievements) != 1 || bullet.Achievements[0].ID != e.pos.Achievements[0].ID {
			t.Errorf("achievements = %+v, want the cited one", bullet.Achievements)
		}
	})
}

func TestGetDraftBase(t *testing.T) {
	t.Run("returns the base CV's bullets as they were at generation beside the Draft's", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		d := e.draft(t, id)

		if d.Base == nil || d.Content == nil {
			t.Fatalf("GetDraft() base = %+v, content = %+v, want both", d.Base, d.Content)
		}
		wantBase := []dto.DraftBullet{
			{Text: "Built and maintained the public APIs for the platform", AchievementIDs: []string{}},
			{Text: "Ran on-call", AchievementIDs: []string{}},
			{Text: "Wrote docs", AchievementIDs: []string{}},
		}
		if diff := cmp.Diff(wantBase, d.Base.Positions[0].Bullets); diff != "" {
			t.Errorf("base bullets (-want +got):\n%s", diff)
		}
		want := []dto.DraftBullet{{Text: "Cut p99 latency by quickly moving queries to Postgres", AchievementIDs: []string{e.pos.Achievements[0].ID}}}
		if diff := cmp.Diff(want, d.Content.Positions[0].Bullets); diff != "" {
			t.Errorf("content bullets (-want +got):\n%s", diff)
		}
	})
}

func TestGetDraftSkillGroups(t *testing.T) {
	t.Run("a Draft that did not edit Skills shows the base Skill Lines", func(t *testing.T) {
		e, id, _ := renumberedDraft(t)

		d := e.draft(t, id)

		if d.Base == nil || d.Content == nil {
			t.Fatalf("GetDraft() base = %+v, content = %+v, want both", d.Base, d.Content)
		}
		want := []dto.SkillGroup{{Items: []string{"Go", "SQL"}}}
		if diff := cmp.Diff(want, d.Base.SkillGroups); diff != "" {
			t.Errorf("base skill groups (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(want, d.Content.SkillGroups); diff != "" {
			t.Errorf("content skill groups (-want +got):\n%s", diff)
		}
	})
}

func TestKeepDraft(t *testing.T) {
	t.Run("queues at once and the next tick rebuilds the Draft as a child Tab of the base Doc", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		ctx := t.Context()
		id := e.readyDrafts(t, 1)[0]

		queued, err := e.svc.KeepDraft(ctx, userID, dto.TailoringDraftQuery{ID: id})
		if err != nil || queued.Status != "keeping" || queued.Outcome != nil {
			t.Fatalf("KeepDraft() = %+v, %v, want keeping and undecided", queued, err)
		}
		e.run(t, tick{})

		got := e.draft(t, id)
		if got.Status != "ready" || got.Outcome == nil || *got.Outcome != dto.OutcomeKept || got.KeptAs != "tab" || got.KeepNote != "" {
			t.Errorf("GetDraft() = %+v, want ready, kept as a tab", got)
		}
		wantURL := "https://docs.google.com/document/d/" + docID + "/edit?tab=t.added1"
		if got.DraftDocID != docID || got.DraftDocURL == nil || *got.DraftDocURL != wantURL {
			t.Errorf("GetDraft() doc = %q, url = %v, want %q, %q", got.DraftDocID, got.DraftDocURL, docID, wantURL)
		}
		if diff := cmp.Diff([]string{"Acme \u2014 Platform Engineer"}, e.drive.AddedTabs); diff != "" {
			t.Errorf("added tabs (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{tabID}, e.drive.ChildOf); diff != "" {
			t.Errorf("tab parents (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"copy-1"}, e.drive.Deleted); diff != "" {
			t.Errorf("deleted files (-want +got):\n%s", diff)
		}
		if len(e.drive.Renamed) != 0 {
			t.Errorf("renamed Docs = %v, want none", e.drive.Renamed)
		}
		if _, err := e.svc.KeepDraft(ctx, userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Errorf("KeepDraft() again err = %v, want it idempotent", err)
		}
	})

	t.Run("editing a draft kept as a Tab writes to that Tab", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		e.run(t, tick{})
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID

		svc := cvtailor.NewService(e.store, cvtailortest.Docs{TabJSON: baseTab(t)}, e.asker, e.drive)
		e.saved(t, svc, id, dto.SlotEdit{SlotID: slot, Text: "Cut latency"})

		last := e.drive.Updates[len(e.drive.Updates)-1]
		if len(last) == 0 {
			t.Fatal("no Docs write after the edit")
		}
		for _, raw := range last {
			if !strings.Contains(string(raw), `"tabId":"t.added1"`) {
				t.Errorf("edit request %s does not target the kept tab", raw)
			}
		}
	})

	t.Run("without Docs write access the Doc is renamed and the reason shown", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		e.drive.AddTabErr = google.ErrNoWriteAccess

		e.run(t, tick{})

		got := e.draft(t, id)
		if got.KeptAs != "doc" || got.KeepNote == "" || got.DraftDocID != "copy-1" {
			t.Errorf("GetDraft() = %+v, want kept as the renamed Doc with a note", got)
		}
		want := map[string]string{"copy-1": "Acme \u2014 Platform Engineer"}
		if diff := cmp.Diff(want, e.drive.Renamed); diff != "" {
			t.Errorf("renamed Docs (-want +got):\n%s", diff)
		}
	})

	t.Run("a Tab that renders differently is deleted and the Doc kept instead", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		e.drive.TabPages = map[string]int{"t.added1": 2}

		e.run(t, tick{})

		got := e.draft(t, id)
		if got.KeptAs != "doc" || got.KeepNote == "" || got.DraftDocID != "copy-1" {
			t.Errorf("GetDraft() = %+v, want kept as the Doc with a note", got)
		}
		last := e.drive.Updates[len(e.drive.Updates)-1]
		if len(last) != 1 || !strings.Contains(string(last[0]), `"deleteTab":{"tabId":"t.added1"}`) {
			t.Errorf("last Docs write = %s, want the kept tab deleted", last)
		}
	})

	t.Run("a Tab Google rejects falls back to the Doc on the same attempt", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		rejected := &google.RejectedError{Status: http.StatusBadRequest, Message: "Invalid requests[3]"}

		e.run(t, tick{drive: cvtailortest.FailsFirstBatchUpdate(e.drive, rejected)})

		got := e.draft(t, id)
		if got.KeptAs != "doc" || got.Outcome == nil || *got.Outcome != dto.OutcomeKept ||
			!strings.Contains(got.KeepNote, "Invalid requests[3]") || got.DraftDocID != "copy-1" {
			t.Errorf("GetDraft() = %+v, want kept as the Doc with Google's reason", got)
		}
		last := e.drive.Updates[len(e.drive.Updates)-1]
		if len(last) != 1 || !strings.Contains(string(last[0]), `"deleteTab":{"tabId":"t.added1"}`) {
			t.Errorf("last Docs write = %s, want the kept tab deleted", last)
		}
	})

	t.Run("a Tab left half-built by a crashed attempt is deleted on retry", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		e.store.LeaveHalfBuiltTab(id, "t.half")

		e.run(t, tick{})

		deleted := false
		for _, batch := range e.drive.Updates {
			deleted = deleted || (len(batch) == 1 && strings.Contains(string(batch[0]), `"tabId":"t.half"`))
		}
		if !deleted {
			t.Errorf("Docs writes = %v, want the half-built tab deleted", e.drive.Updates)
		}
	})

	t.Run("the keep records what happened to every bank Achievement", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		onCV := addAchievements(t, e.pos.ID, e.store, "Ran on-call")[0]
		e.asker.answers = map[string]dto.Answer{
			questionPrefix + e.pos.Achievements[0].Text: {PYes: 0.4, PNo: 0.1},
			questionPrefix + e.pos.Achievements[1].Text: {PYes: 0.7, PNo: 0.1},
		}
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}

		e.run(t, tick{})

		yes0, no, yes1 := 0.4, 0.1, 0.7
		want := []dto.AchievementOutcome{
			{AchievementID: e.pos.Achievements[0].ID, PYes: &yes0, PNo: &no, InFinalCV: true, Reworded: true},
			{AchievementID: e.pos.Achievements[1].ID, PYes: &yes1, PNo: &no, OutscoredKeptButLeftOut: true},
			{AchievementID: onCV.ID, OnSourceCV: true},
		}
		if diff := cmp.Diff(want, e.store.AchievementOutcomes(id)); diff != "" {
			t.Errorf("outcomes (-want +got):\n%s", diff)
		}
	})

	t.Run("the keep still records outcomes when Jev cannot answer", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		e.asker.err = errors.New("jev down")
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}

		e.run(t, tick{})

		want := []dto.AchievementOutcome{
			{AchievementID: e.pos.Achievements[0].ID, InFinalCV: true, Reworded: true},
			{AchievementID: e.pos.Achievements[1].ID},
		}
		if diff := cmp.Diff(want, e.store.AchievementOutcomes(id)); diff != "" {
			t.Errorf("outcomes (-want +got):\n%s", diff)
		}
	})

	t.Run("discarding a Draft records no outcomes", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		if _, err := e.svc.DiscardDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		e.run(t, tick{})

		if got := e.store.AchievementOutcomes(id); len(got) != 0 {
			t.Errorf("outcomes = %+v, want none", got)
		}
	})

	t.Run("a second Draft of a Job conflicts until the first is discarded", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := t.Context()
		first := dto.TailoringDraftQuery{ID: e.readyDrafts(t, 1)[0]}
		if _, err := e.svc.KeepDraft(ctx, userID, first); err != nil {
			t.Fatal(err)
		}
		e.run(t, tick{})
		second := dto.TailoringDraftQuery{ID: e.readyDrafts(t, 1)[0]}

		_, err := e.svc.KeepDraft(ctx, userID, second)
		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(second)", err, apperr.KindConflict)
		}

		if _, err := e.svc.DiscardDraft(ctx, userID, first); err != nil {
			t.Fatal(err)
		}
		if _, err := e.svc.KeepDraft(ctx, userID, second); err != nil {
			t.Errorf("KeepDraft(second) after discarding the first err = %v, want nil", err)
		}
	})

	t.Run("a Draft that is not ready conflicts", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(pending)", err, apperr.KindConflict)
		}
	})

	t.Run("a Draft being kept cannot be discarded", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}

		_, err := e.svc.DiscardDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "DiscardDraft(keeping)", err, apperr.KindConflict)
		}
	})

	t.Run("repeated Drive failures leave the Draft ready, unkept, with the error", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		drive := &cvtailortest.Drive{AddTabErr: google.ErrNoWriteAccess, RenameErr: errors.New("drive down")}

		for range dto.MaxDraftAttempts {
			e.store.MakeDraftDue(id)
			e.run(t, tick{drive: drive})
		}

		got := e.draft(t, id)
		if got.Status != "ready" || got.Outcome != nil || !strings.Contains(got.LastError, "drive down") {
			t.Errorf("GetDraft() = %+v, want ready, unkept, last error set", got)
		}
	})
}

func TestDiscardDraft(t *testing.T) {
	t.Run("a draft kept as a Tab deletes only its Tab, never the base Doc", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		e.run(t, tick{})
		e.drive.Deleted = nil

		if _, err := e.svc.DiscardDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}

		last := e.drive.Updates[len(e.drive.Updates)-1]
		if len(e.drive.Deleted) != 0 || len(last) != 1 || !strings.Contains(string(last[0]), `"deleteTab":{"tabId":"t.added1"}`) {
			t.Errorf("deleted files = %v, last Docs write = %s, want only the kept tab deleted", e.drive.Deleted, last)
		}
	})

	t.Run("deletes the Drive file and keeps the row", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := t.Context()
		id := e.readyDrafts(t, 1)[0]

		d, err := e.svc.DiscardDraft(ctx, userID, dto.TailoringDraftQuery{ID: id})
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff([]string{"copy-1"}, e.drive.Deleted); diff != "" {
			t.Errorf("deleted files (-want +got):\n%s", diff)
		}
		if d.Outcome == nil || *d.Outcome != dto.OutcomeDiscarded || d.DraftDocURL != nil {
			t.Errorf("DiscardDraft() = %+v, want discarded with no Doc URL", d)
		}
		if got := e.draft(t, id); got.ID != id {
			t.Errorf("GetDraft() = %+v, want the row to remain", got)
		}
		_, err = e.svc.KeepDraft(ctx, userID, dto.TailoringDraftQuery{ID: id})
		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(discarded)", err, apperr.KindConflict)
		}
	})

	t.Run("a Draft that is not ready conflicts", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.DiscardDraft(t.Context(), userID, dto.TailoringDraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "DiscardDraft(pending)", err, apperr.KindConflict)
		}
	})
}

func TestDraftPDF(t *testing.T) {
	t.Run("a Draft that is not ready is not found", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.DraftPDF(t.Context(), userID, dto.TailoringDraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("%s err = %v, want kind %v", "DraftPDF(pending)", err, apperr.KindNotFound)
		}
	})
}
