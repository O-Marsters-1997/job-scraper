package cvtailor_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func (e draftEnv) readyDrafts(t *testing.T, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		ids[i] = e.queue(t)
	}
	editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency by moving Redis queries to Postgres", 0))
	for range n {
		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, e.drive, editor, apiKey)
	}
	return ids
}

func (e draftEnv) router(drive cvtailor.Drive) chi.Router {
	r := chi.NewRouter()
	cvtailor.Build(cvtailor.Deps{Store: e.store, Drive: drive}).Routes(r)
	return r
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
			{Text: "Cut p99 latency by moving ", Novel: false},
			{Text: "Redis", Novel: true},
			{Text: " queries to Postgres", Novel: false},
		}
		if diff := cmp.Diff(want, bullet.Segments); diff != "" {
			t.Errorf("segments (-want +got):\n%s", diff)
		}
		if len(bullet.Achievements) != 1 || bullet.Achievements[0].ID != e.pos.Achievements[0].ID {
			t.Errorf("achievements = %+v, want the cited one", bullet.Achievements)
		}
	})
}

func TestKeepDraft(t *testing.T) {
	t.Run("a second Draft of a Job conflicts until the first is discarded", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := context.Background()
		ids := e.readyDrafts(t, 2)
		first, second := dto.DraftQuery{ID: ids[0]}, dto.DraftQuery{ID: ids[1]}

		kept, err := e.svc.KeepDraft(ctx, user, first)
		if err != nil || kept.Outcome == nil || *kept.Outcome != dto.OutcomeKept || kept.DraftDocURL == nil {
			t.Fatalf("KeepDraft() = %+v, %v, want kept with its Doc URL", kept, err)
		}
		if _, err := e.svc.KeepDraft(ctx, user, first); err != nil {
			t.Errorf("KeepDraft() again err = %v, want it idempotent", err)
		}
		_, err = e.svc.KeepDraft(ctx, user, second)
		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(second)", err, apperr.KindConflict)
		}

		if _, err := e.svc.DiscardDraft(ctx, user, first); err != nil {
			t.Fatal(err)
		}
		if _, err := e.svc.KeepDraft(ctx, user, second); err != nil {
			t.Errorf("KeepDraft(second) after discarding the first err = %v, want nil", err)
		}
	})

	t.Run("a Draft that is not ready conflicts", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)

		_, err := e.svc.KeepDraft(context.Background(), user, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(pending)", err, apperr.KindConflict)
		}
	})
}

func TestDiscardDraft(t *testing.T) {
	t.Run("deletes the Drive file and keeps the row", func(t *testing.T) {
		e := newDraftEnv(t)
		ctx := context.Background()
		id := e.readyDrafts(t, 1)[0]

		d, err := e.svc.DiscardDraft(ctx, user, dto.DraftQuery{ID: id})
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
		_, err = e.svc.KeepDraft(ctx, user, dto.DraftQuery{ID: id})
		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "KeepDraft(discarded)", err, apperr.KindConflict)
		}
	})

	t.Run("a Draft that is not ready conflicts", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)

		_, err := e.svc.DiscardDraft(context.Background(), user, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("%s err = %v, want kind %v", "DiscardDraft(pending)", err, apperr.KindConflict)
		}
	})
}

func TestDraftPDF(t *testing.T) {
	t.Run("a Draft that is not ready is not found", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)

		_, err := e.svc.DraftPDF(context.Background(), user, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("%s err = %v, want kind %v", "DraftPDF(pending)", err, apperr.KindNotFound)
		}
	})
}

func TestReviewRoutes(t *testing.T) {
	t.Run("streams the PDF of a ready Draft", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		r := e.router(cvtailortest.ExportsPDF(e.drive, "%PDF-fake"))

		w := do(t, r, http.MethodGet, "/tailoring/drafts/"+id+"/pdf", "")

		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/pdf" || w.Body.String() != "%PDF-fake" {
			t.Errorf("GET pdf = %d %q %q, want the streamed PDF", w.Code, w.Header().Get("Content-Type"), w.Body)
		}
	})

	t.Run("keeps, lists and discards a Job's Drafts", func(t *testing.T) {
		e := newDraftEnv(t)
		r := e.router(e.drive)
		ids := e.readyDrafts(t, 2)

		if w := do(t, r, http.MethodPost, "/tailoring/drafts/"+ids[0]+"/keep", ""); w.Code != http.StatusOK {
			t.Fatalf("POST keep status = %d, body %s", w.Code, w.Body)
		}
		if w := do(t, r, http.MethodPost, "/tailoring/drafts/"+ids[1]+"/keep", ""); w.Code != http.StatusConflict {
			t.Errorf("POST keep of a second Draft status = %d, want 409", w.Code)
		}

		w := do(t, r, http.MethodGet, "/tailoring/jobs/"+jobID+"/drafts", "")
		var list []dto.Draft
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || w.Code != http.StatusOK {
			t.Fatalf("GET job drafts = %d %s, %v", w.Code, w.Body, err)
		}
		var gotIDs []string
		for _, d := range list {
			gotIDs = append(gotIDs, d.ID)
		}
		slices.Sort(gotIDs)
		if diff := cmp.Diff(ids, gotIDs); diff != "" {
			t.Errorf("listed Draft ids (-want +got):\n%s", diff)
		}

		if w := do(t, r, http.MethodPost, "/tailoring/drafts/"+ids[0]+"/discard", ""); w.Code != http.StatusOK {
			t.Errorf("POST discard status = %d, body %s", w.Code, w.Body)
		}
	})
}
