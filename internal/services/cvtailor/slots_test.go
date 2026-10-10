package cvtailor_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
)

func (e draftEnv) editing(t *testing.T, drive cvtailor.Drive) *cvtailor.Service {
	t.Helper()
	return cvtailor.NewService(e.store, cvtailortest.Docs{TabJSON: baseTab(t)}, nil, drive)
}

func profileTab(t *testing.T) cvtailortest.Docs {
	t.Helper()
	return cvtailortest.Docs{TabJSON: tabJSON(t, head("Profile"), prose("Engineer who ships."), head(heading), bullet("Built and maintained the public APIs for the platform"))}
}

func (e draftEnv) readyWithProfile(t *testing.T, profile string) (string, *cvtailor.Service) {
	t.Helper()
	docs := profileTab(t)
	result := e.bulletResult("Cut p99 latency by moving queries to Postgres", 0)
	result.Edits.Profile = &profile
	id := e.queue(t)
	e.run(t, tick{docs: docs, editor: cvtailortest.Editing(result)})
	return id, cvtailor.NewService(e.store, docs, nil, e.drive)
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
		if bullet.SlotID != slot {
			t.Errorf("SaveDraftSlots().Provenance bullet slot = %q, want %q", bullet.SlotID, slot)
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

	t.Run("saves an edit longer than the base text without a length finding", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		long := "Cut p99 latency by moving queries to Postgres and then tuning every index across the whole platform for good measure"

		got := e.saved(t, e.editing(t, e.drive), id, dto.SlotEdit{SlotID: slot, Text: long})

		for _, f := range got.Findings {
			if f.Check == "slot_length" {
				t.Errorf("SaveDraftSlots().Findings = %+v, want no slot_length finding", got.Findings)
			}
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

	t.Run("writes an edited profile to the Doc and its own provenance", func(t *testing.T) {
		e := newDraftEnv(t)
		id, svc := e.readyWithProfile(t, "Engineer who ships.")

		got := e.saved(t, svc, id, dto.SlotEdit{SlotID: "profile", Text: " Engineer who cuts latency. "})

		if inserted := insertedText(t, e.drive); !slices.Contains(inserted, "Engineer who cuts latency.") {
			t.Errorf("inserted text = %q, want it to include the edited profile", inserted)
		}
		if got.Provenance.Profile == nil || got.Provenance.Profile.SlotID != "profile" {
			t.Fatalf("SaveDraftSlots().Provenance.Profile = %+v, want the profile slot", got.Provenance.Profile)
		}
		if got.Content.Profile == nil || *got.Content.Profile != "Engineer who cuts latency." {
			t.Errorf("SaveDraftSlots().Content.Profile = %v, want the edited text", got.Content.Profile)
		}
	})

	t.Run("a bullet-only save leaves the profile paragraph alone", func(t *testing.T) {
		e := newDraftEnv(t)
		id, svc := e.readyWithProfile(t, "Engineer who ships.")
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID

		e.saved(t, svc, id, dto.SlotEdit{SlotID: slot, Text: "Cut p99 latency"})

		if diff := cmp.Diff([]string{"Cut p99 latency"}, insertedText(t, e.drive)); diff != "" {
			t.Errorf("inserted text mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("rejects a profile slot when the CV has no profile", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		_, err := e.editing(t, e.drive).SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: id, Slots: []dto.SlotEdit{{SlotID: "profile", Text: "x"}}})

		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("SaveDraftSlots(profile) error = %v, want an invalid error", err)
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

func TestSaveDraftSlotsSkills(t *testing.T) {
	skillsTab := cvtailortest.Docs{TabJSON: baseTab(t, head("Skills"), prose("Languages: Go, SQL"))}
	ready := func(t *testing.T) (draftEnv, string, *cvtailor.Service) {
		t.Helper()
		e := newDraftEnv(t)
		e.bankSkill(t, "Rust")
		res := e.bulletResult("Cut p99 latency", 0.25)
		res.Edits.Skills = []cvedit.SkillGroup{{Label: "Languages", Items: []string{"Go", "SQL"}}}
		id := e.queue(t)
		e.run(t, tick{docs: skillsTab, editor: cvtailortest.Editing(res)})
		return e, id, cvtailor.NewService(e.store, skillsTab, nil, e.drive)
	}
	save := func(t *testing.T, svc *cvtailor.Service, id string, groups ...dto.SkillGroup) (dto.Draft, error) {
		t.Helper()
		return svc.SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: id, Skills: groups})
	}

	t.Run("saves a swap and reorder with the label intact", func(t *testing.T) {
		e, id, svc := ready(t)

		got, err := save(t, svc, id, dto.SkillGroup{Label: "Languages", Items: []string{"Rust", "Go"}})
		if err != nil {
			t.Fatalf("SaveDraftSlots(skills) error = %v", err)
		}

		if diff := cmp.Diff([]dto.SkillGroup{{Label: "Languages", Items: []string{"Rust", "Go"}}}, got.Content.SkillGroups); diff != "" {
			t.Errorf("skill groups mismatch (-want +got):\n%s", diff)
		}
		if inserted := insertedText(t, e.drive); !slices.Contains(inserted, "Rust, Go") {
			t.Errorf("inserted = %q, want the items \"Rust, Go\" written", inserted)
		}
		if blocks := findingChecks(got.Findings, "block"); len(blocks) != 0 {
			t.Errorf("block findings = %v, want none", blocks)
		}
		if !got.SkillsEditable {
			t.Error("SaveDraftSlots().SkillsEditable = false, want true")
		}
	})

	t.Run("stores a skill with the CV's or Bank's own casing", func(t *testing.T) {
		_, id, svc := ready(t)

		got, err := save(t, svc, id, dto.SkillGroup{Label: "Languages", Items: []string{"rust", "GO"}})
		if err != nil {
			t.Fatalf("SaveDraftSlots(skills) error = %v", err)
		}

		if diff := cmp.Diff([]string{"Rust", "Go"}, got.Content.SkillGroups[0].Items); diff != "" {
			t.Errorf("items mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("rejects skills on a Draft stored with flat skills", func(t *testing.T) {
		e, id, svc := ready(t)
		legacy := json.RawMessage(`{"positions":[],"skills":["Go","SQL"]}`)
		if err := e.store.SetDraftEdits(t.Context(), userID, id, legacy, nil); err != nil {
			t.Fatalf("SetDraftEdits() error = %v", err)
		}

		_, err := save(t, svc, id, dto.SkillGroup{Label: "Languages", Items: []string{"Rust", "Go"}})

		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("SaveDraftSlots(skills on legacy) error = %v, want an invalid error", err)
		}
	})

	rejected := []struct {
		name   string
		groups []dto.SkillGroup
	}{
		{"an item that is neither a base item nor a Bank Skill", []dto.SkillGroup{{Label: "Languages", Items: []string{"Perl", "Go"}}}},
		{"a changed label", []dto.SkillGroup{{Label: "Tools", Items: []string{"Go", "SQL"}}}},
		{"a changed line count", []dto.SkillGroup{{Label: "Languages", Items: []string{"Go"}}, {Items: []string{"SQL"}}}},
		{"a duplicate item", []dto.SkillGroup{{Label: "Languages", Items: []string{"Go", "go"}}}},
	}
	for _, tc := range rejected {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			_, id, svc := ready(t)

			_, err := save(t, svc, id, tc.groups...)

			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("SaveDraftSlots(%+v) error = %v, want an invalid error", tc.groups, err)
			}
		})
	}
}
