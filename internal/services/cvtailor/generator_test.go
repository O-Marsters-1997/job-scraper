package cvtailor_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

const apiKey = cvtailortest.Key("sk-or-test")

func newDrive() *cvtailortest.Drive {
	return &cvtailortest.Drive{Tabs: []google.Tab{{ID: tabID}, {ID: "t.1"}}}
}

func baseTab(t *testing.T, extra ...cvLine) json.RawMessage {
	t.Helper()
	lines := []cvLine{head(heading), bullet("Built and maintained the public APIs for the platform"), bullet("Ran on-call"), bullet("Wrote docs")}
	return tabJSON(t, append(lines, extra...)...)
}

func (e draftEnv) queue(t *testing.T) string {
	t.Helper()
	ref, err := e.svc.CreateDraft(t.Context(), userID, e.input)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	return ref.ID
}

func (e draftEnv) queueOnOtherJob(t *testing.T) string {
	t.Helper()
	other := e
	other.input.JobID = "job-other"
	return other.queue(t)
}

func newQueuedDraft(t *testing.T) (draftEnv, string) {
	t.Helper()
	e := newDraftEnv(t)
	return e, e.queue(t)
}

func (e draftEnv) draft(t *testing.T, id string) dto.Draft {
	t.Helper()
	d, err := e.svc.GetDraft(t.Context(), userID, dto.DraftQuery{ID: id})
	if err != nil {
		t.Fatalf("GetDraft(%s) error = %v", id, err)
	}
	return d
}

type tick struct {
	docs  cvtailor.DocFetcher
	drive cvtailor.Drive
}

func (e draftEnv) run(t *testing.T, tk tick) {
	t.Helper()
	if tk.docs == nil {
		tk.docs = cvtailortest.Docs{TabJSON: baseTab(t)}
	}
	if tk.drive == nil {
		tk.drive = e.drive
	}
	m := cvtailor.Build(cvtailor.Deps{Store: e.store, Docs: tk.docs, Drive: tk.drive, Asker: e.asker, Creds: apiKey})
	if err := m.RunTick(t.Context()); err != nil {
		t.Fatalf("RunTick() error = %v", err)
	}
}

func findingChecks(findings []dto.DraftFinding, severity string) []string {
	var out []string
	for _, f := range findings {
		if f.Severity == severity {
			out = append(out, f.Check)
		}
	}
	return out
}

func (e draftEnv) citedEdit(text string) cvedit.EditSet {
	return cvedit.EditSet{Positions: []cvedit.PositionEdit{{
		PositionID: e.pos.ID,
		Bullets:    []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: text}},
	}}}
}

func (e draftEnv) readyWith(t *testing.T, tk tick, edits cvedit.EditSet, chosen ...string) string {
	t.Helper()
	id := e.queue(t)
	e.run(t, tk)
	var generated cvedit.EditSet
	if err := json.Unmarshal(e.store.DraftResult(id).EditSet, &generated); err != nil {
		t.Fatal(err)
	}
	slotIDs := map[string][]string{}
	for _, pe := range generated.Positions {
		for _, b := range pe.Bullets {
			slotIDs[pe.PositionID] = append(slotIDs[pe.PositionID], b.SlotID)
		}
	}
	for _, pe := range edits.Positions {
		for i := range pe.Bullets {
			pe.Bullets[i].SlotID = slotIDs[pe.PositionID][i]
			for _, aid := range pe.Bullets[i].AchievementIDs {
				if !slices.Contains(chosen, aid) {
					chosen = append(chosen, aid)
				}
			}
		}
	}
	raw, err := json.Marshal(edits)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.store.SetDraftEdits(t.Context(), userID, id, raw, nil, chosen); err != nil {
		t.Fatalf("SetDraftEdits() error = %v", err)
	}
	return id
}

func TestGeneratorRunTick(t *testing.T) {
	t.Run("builds a ready Draft from a trimmed copy of the base tab", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		e.run(t, tick{})

		url := "https://docs.google.com/document/d/copy-1/edit"
		wantDraft := dto.Draft{ID: id, JobID: jobID, Mode: dto.DraftModeGenerate, Status: "ready", DraftDocURL: &url, DraftDocID: "copy-1"}
		if diff := cmp.Diff(wantDraft, e.draft(t, id), cmpopts.IgnoreFields(dto.Draft{}, "Findings", "CreatedAt", "EditSet", "Provenance", "Content", "Base", "BaseContent", "BaseDocID", "BaseTabID", "DraftTabID", "AchievementIDs")); diff != "" {
			t.Errorf("GetDraft(%s) mismatch (-want +got):\n%s", id, diff)
		}
		wantResult := dto.DraftResult{JobFingerprint: "fp-1", DraftDocID: "copy-1"}
		if diff := cmp.Diff(wantResult, e.store.DraftResult(id), cmpopts.IgnoreFields(dto.DraftResult{}, "EditSet", "BaseContent", "Findings")); diff != "" {
			t.Errorf("recorded result mismatch (-want +got):\n%s", diff)
		}
		if len(e.drive.Updates) != 1 {
			t.Fatalf("Docs writes = %d batches, want only the tab trim", len(e.drive.Updates))
		}
		var trim []map[string]map[string]string
		for _, raw := range e.drive.Updates[0] {
			trim = append(trim, handlerstest.DecodeJSON[map[string]map[string]string](t, raw))
		}
		wantTrim := []map[string]map[string]string{{"deleteTab": {"tabId": "t.1"}}}
		if diff := cmp.Diff(wantTrim, trim); diff != "" {
			t.Errorf("tab trim mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("an as-is Draft copies the base tab unchanged", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		e.input.Mode = dto.DraftModeAsIs
		id := e.queue(t)

		e.run(t, tick{})

		d := e.draft(t, id)
		wantBullets := []dto.DraftBullet{
			{Text: "Built and maintained the public APIs for the platform", AchievementIDs: []string{}},
			{Text: "Ran on-call", AchievementIDs: []string{}},
			{Text: "Wrote docs", AchievementIDs: []string{}},
		}
		if d.Status != "ready" || d.Content == nil || len(d.Content.Positions) != 1 {
			t.Fatalf("GetDraft(%s) = %+v, want a ready Draft with one Position", id, d)
		}
		if diff := cmp.Diff(wantBullets, d.Content.Positions[0].Bullets); diff != "" {
			t.Errorf("Draft bullets mismatch (-want +got):\n%s", diff)
		}
		if len(e.drive.Updates) != 1 {
			t.Errorf("Docs writes = %d batches, want only the tab trim", len(e.drive.Updates))
		}
	})

	t.Run("a generate Draft replaces only low-lean bullets", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		addAchievements(t, e.pos.ID, e.store, "Ran on-call", "Led the incident review process")
		e.asker.answers = map[string]dto.Answer{
			"Would this job value a candidate who: Ran on-call":                     {PNo: 0.6},
			"Would this job value a candidate who: Led the incident review process": {PYes: 0.7},
		}
		id := e.queue(t)

		e.run(t, tick{})

		d := e.draft(t, id)
		if d.Status != "ready" || d.Content == nil || len(d.Content.Positions) != 1 {
			t.Fatalf("GetDraft(%s) = %+v, want a ready Draft with one Position", id, d)
		}
		var got []string
		for _, b := range d.Content.Positions[0].Bullets {
			got = append(got, b.Text)
		}
		want := []string{"Built and maintained the public APIs for the platform", "Led the incident review process", "Wrote docs"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Draft bullets mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("deletes the copy when a Docs write fails", func(t *testing.T) {
		t.Parallel()
		e, id := newQueuedDraft(t)

		e.run(t, tick{drive: cvtailortest.FailsBatchUpdate(e.drive, context.DeadlineExceeded)})

		if diff := cmp.Diff([]string{"copy-1"}, e.drive.Deleted); diff != "" {
			t.Errorf("deleted files mismatch (-want +got):\n%s", diff)
		}
		d := e.draft(t, id)
		if d.Status != "pending" || d.DraftDocURL != nil || d.DraftDocID != "" || !strings.Contains(d.LastError, context.DeadlineExceeded.Error()) {
			t.Errorf("GetDraft(%s) = %+v, want a retry pending with %q and no Doc", id, d, context.DeadlineExceeded)
		}
	})

	t.Run("reads a Draft stored with flat skills", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		e.run(t, tick{docs: cvtailortest.Docs{TabJSON: baseTab(t, head("Skills"), prose("Go, SQL"))}})
		legacy := json.RawMessage(`{"positions":[],"skills":["SQL","Go"]}`)
		if err := e.store.SetDraftEdits(t.Context(), userID, id, legacy, nil, nil); err != nil {
			t.Fatalf("SetDraftEdits() error = %v", err)
		}

		d := e.draft(t, id)

		if diff := cmp.Diff([]string{"SQL", "Go"}, d.Content.Skills); diff != "" {
			t.Errorf("content skills mismatch (-want +got):\n%s", diff)
		}
		if d.SkillsEditable {
			t.Error("GetDraft(legacy).SkillsEditable = true, want false")
		}
	})
}
