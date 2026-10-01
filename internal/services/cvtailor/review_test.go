package cvtailor_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func (e draftEnv) readyDrafts(t *testing.T, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		ids[i] = e.queue(t)
	}
	editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency by quickly moving queries to Postgres", 0))
	for range n {
		e.run(t, tick{editor: editor})
	}
	return ids
}

func TestGetDraftProvenance(t *testing.T) {
	t.Run("marks bullet words absent from the cited achievements as novel", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		d := e.draft(t, id)

		if d.Provenance == nil || len(d.Provenance.Positions) != 1 || len(d.Provenance.Positions[0].Bullets) != 1 {
			t.Fatalf("GetDraft().Provenance = %+v, want one position with one bullet", d.Provenance)
		}
		bullet := d.Provenance.Positions[0].Bullets[0]
		want := []dto.TextSegment{
			{Text: "Cut p99 latency by ", Novel: false},
			{Text: "quickly", Novel: true},
			{Text: " moving queries to Postgres", Novel: false},
		}
		if diff := cmp.Diff(want, bullet.Segments); diff != "" {
			t.Errorf("segments (-want +got):\n%s", diff)
		}
		if len(bullet.Achievements) != 1 || bullet.Achievements[0].ID != e.pos.Achievements[0].ID {
			t.Errorf("achievements = %+v, want the cited one", bullet.Achievements)
		}
	})

	t.Run("shows a kept bullet as the user's own text", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		kept := cvedit.Result{Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{
			PositionID: e.pos.ID,
			Bullets:    []cvedit.Bullet{{Keep: true, Text: "Built and maintained the public APIs for the platform"}},
		}}}}
		e.run(t, tick{editor: cvtailortest.Editing(kept)})

		d := e.draft(t, id)

		if d.Provenance == nil || len(d.Provenance.Positions) != 1 || len(d.Provenance.Positions[0].Bullets) != 1 {
			t.Fatalf("GetDraft().Provenance = %+v, want one position with one bullet", d.Provenance)
		}
		want := []dto.TextSegment{{Text: "Built and maintained the public APIs for the platform"}}
		if diff := cmp.Diff(want, d.Provenance.Positions[0].Bullets[0].Segments); diff != "" {
			t.Errorf("segments (-want +got):\n%s", diff)
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

func TestKeepDraft(t *testing.T) {
	t.Run("queues at once and the next tick renames the Doc and marks it kept", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := t.Context()
		id := e.readyDrafts(t, 1)[0]

		queued, err := e.svc.KeepDraft(ctx, userID, dto.DraftQuery{ID: id})
		if err != nil || queued.Status != "keeping" || queued.Outcome != nil {
			t.Fatalf("KeepDraft() = %+v, %v, want keeping and undecided", queued, err)
		}
		e.run(t, tick{})

		got := e.draft(t, id)
		if got.Status != "ready" || got.Outcome == nil || *got.Outcome != dto.OutcomeKept || got.KeptAs != "doc" {
			t.Errorf("GetDraft() = %+v, want ready, kept as a doc", got)
		}
		want := map[string]string{"copy-1": "Acme \u2014 Platform Engineer"}
		if diff := cmp.Diff(want, e.drive.Renamed); diff != "" {
			t.Errorf("renamed Docs (-want +got):\n%s", diff)
		}
		if _, err := e.svc.KeepDraft(ctx, userID, dto.DraftQuery{ID: id}); err != nil {
			t.Errorf("KeepDraft() again err = %v, want it idempotent", err)
		}
	})

	t.Run("a second Draft of a Job conflicts until the first is discarded", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := t.Context()
		ids := e.readyDrafts(t, 2)
		first, second := dto.DraftQuery{ID: ids[0]}, dto.DraftQuery{ID: ids[1]}
		if _, err := e.svc.KeepDraft(ctx, userID, first); err != nil {
			t.Fatal(err)
		}

		_, err := e.svc.KeepDraft(ctx, userID, second)
		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(second)", err, apperr.KindConflict)
		}

		e.run(t, tick{})
		if _, err := e.svc.DiscardDraft(ctx, userID, first); err != nil {
			t.Fatal(err)
		}
		if _, err := e.svc.KeepDraft(ctx, userID, second); err != nil {
			t.Errorf("KeepDraft(second) after discarding the first err = %v, want nil", err)
		}
	})

	t.Run("a Draft that is not ready conflicts", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.KeepDraft(t.Context(), userID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(pending)", err, apperr.KindConflict)
		}
	})

	t.Run("a Draft being kept cannot be discarded", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.DraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}

		_, err := e.svc.DiscardDraft(t.Context(), userID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "DiscardDraft(keeping)", err, apperr.KindConflict)
		}
	})

	t.Run("repeated Drive failures leave the Draft ready, unkept, with the error", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.svc.KeepDraft(t.Context(), userID, dto.DraftQuery{ID: id}); err != nil {
			t.Fatal(err)
		}
		drive := &cvtailortest.Drive{RenameErr: errors.New("drive down")}

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
	t.Run("deletes the Drive file and keeps the row", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := t.Context()
		id := e.readyDrafts(t, 1)[0]

		d, err := e.svc.DiscardDraft(ctx, userID, dto.DraftQuery{ID: id})
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
		_, err = e.svc.KeepDraft(ctx, userID, dto.DraftQuery{ID: id})
		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(discarded)", err, apperr.KindConflict)
		}
	})

	t.Run("a Draft that is not ready conflicts", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.DiscardDraft(t.Context(), userID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "DiscardDraft(pending)", err, apperr.KindConflict)
		}
	})
}

func TestDraftPDF(t *testing.T) {
	t.Run("a Draft that is not ready is not found", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		_, err := e.svc.DraftPDF(t.Context(), userID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("%s err = %v, want kind %v", "DraftPDF(pending)", err, apperr.KindNotFound)
		}
	})
}
