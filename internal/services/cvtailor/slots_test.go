package cvtailor_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
)

func (e draftEnv) editing(t *testing.T, drive cvtailor.Drive) *cvtailor.Service {
	t.Helper()
	return cvtailor.NewService(e.store, cvtailortest.Docs{TabJSON: baseTab(t)}, nil, drive)
}

func (e draftEnv) saved(t *testing.T, svc *cvtailor.Service, id string, slots ...dto.SlotEdit) dto.Draft {
	t.Helper()
	d, err := svc.SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: id, Slots: slots})
	if err != nil {
		t.Fatalf("SaveDraftSlots(%s) error = %v", id, err)
	}
	return d
}

func insertedText(t *testing.T, drive *cvtailortest.Drive) []string {
	t.Helper()
	var out []string
	for _, raw := range drive.Updates[len(drive.Updates)-1] {
		if req := handlerstest.DecodeJSON[docedit.Request](t, raw); req.InsertText != nil {
			out = append(out, req.InsertText.Text)
		}
	}
	return out
}

func TestSaveDraftSlots(t *testing.T) {
	t.Run("writes the edited bullet to the Doc and the provenance", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		svc := e.editing(t, e.drive)
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID

		got := e.saved(t, svc, id, dto.SlotEdit{SlotID: slot, Text: " Cut p99 latency by moving queries "})

		if diff := cmp.Diff([]string{"Cut p99 latency by moving queries"}, insertedText(t, e.drive)); diff != "" {
			t.Errorf("inserted text mismatch (-want +got):\n%s", diff)
		}
		bullet := got.Provenance.Positions[0].Bullets[0]
		want := []dto.TextSegment{{Text: "Cut p99 latency by moving queries"}}
		if bullet.SlotID != slot || !slices.Equal(bullet.Segments, want) {
			t.Errorf("SaveDraftSlots().Provenance bullet = %+v, want slot %q with the new text", bullet, slot)
		}
	})

	t.Run("saves an ungrounded edit and reports its finding", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID

		got := e.saved(t, e.editing(t, e.drive), id, dto.SlotEdit{SlotID: slot, Text: "Leveraged Postgres"})

		if len(findingChecks(got.Findings, "block")) == 0 {
			t.Errorf("SaveDraftSlots().Findings = %+v, want a block finding recorded", got.Findings)
		}
	})

	t.Run("saves an edit that adds a page and reports it", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID

		got := e.saved(t, e.editing(t, cvtailortest.ExportsPages(e.drive, 1, 2)), id, dto.SlotEdit{SlotID: slot, Text: "Cut p99 latency"})

		if diff := cmp.Diff([]string{"page_count"}, findingChecks(got.Findings, "block")); diff != "" {
			t.Errorf("block checks mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("rejects an unknown slot or blank text", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		for _, edit := range []dto.SlotEdit{{SlotID: "nope", Text: "x"}, {SlotID: slot, Text: "  "}} {
			_, err := e.editing(t, e.drive).SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: id, Slots: []dto.SlotEdit{edit}})

			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("SaveDraftSlots(%+v) error = %v, want an invalid error", edit, err)
			}
		}
	})

	t.Run("refuses a draft that is not ready or was discarded", func(t *testing.T) {
		e := newDraftEnv(t)
		discarded := e.readyDrafts(t, 1)[0]
		pending := e.queue(t)
		if _, err := e.svc.DiscardDraft(t.Context(), userID, dto.DraftQuery{ID: discarded}); err != nil {
			t.Fatal(err)
		}
		for name, id := range map[string]string{"pending": pending, "discarded": discarded} {
			_, err := e.editing(t, e.drive).SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: id})

			if !apperr.IsKind(err, apperr.KindConflict) {
				t.Errorf("SaveDraftSlots(%s draft) error = %v, want a conflict", name, err)
			}
		}
	})

	t.Run("another user's draft is not found", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		_, err := e.editing(t, e.drive).SaveDraftSlots(t.Context(), otherUserID, dto.DraftSlotsInput{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("SaveDraftSlots(%s) as another user error = %v, want a not-found error", id, err)
		}
	})
}

func TestSaveDraftSlotsRoute(t *testing.T) {
	e := newDraftEnv(t)
	id := e.readyDrafts(t, 1)[0]
	slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
	r := newRouter(cvtailor.Deps{Store: e.store, Drive: e.drive, Docs: cvtailortest.Docs{TabJSON: baseTab(t)}})

	got := handlerstest.Do[dto.Draft](t, r, http.StatusOK, "PUT /tailoring/drafts/"+id+"/slots",
		`{"id":"someone-else","slots":[{"slotId":"`+slot+`","text":"Cut p99 latency"}]}`)

	if got.ID != id {
		t.Errorf("PUT slots Draft id = %q, want the path id %q, not the body's", got.ID, id)
	}
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "PUT /tailoring/drafts/missing/slots", `{"slots":[]}`)
}
