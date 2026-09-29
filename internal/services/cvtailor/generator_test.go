package cvtailor_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

func newDrive() *cvtailortest.Drive {
	return &cvtailortest.Drive{Tabs: []google.Tab{{ID: tabID}, {ID: "t.1"}}}
}

func baseTab(t *testing.T) json.RawMessage {
	t.Helper()
	return tabJSON(t, head(heading), bullet("Built and maintained the public APIs for the platform"), bullet("Ran on-call"), bullet("Wrote docs"))
}

func (e draftEnv) queue(t *testing.T) string {
	t.Helper()
	ref, err := e.svc.CreateDraft(context.Background(), user, e.input)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	return ref.ID
}

func (e draftEnv) draft(t *testing.T, id string) dto.Draft {
	t.Helper()
	d, err := e.svc.GetDraft(context.Background(), user, dto.DraftQuery{ID: id})
	if err != nil {
		t.Fatalf("GetDraft(%s) error = %v", id, err)
	}
	return d
}

func (e draftEnv) run(t *testing.T, docs cvtailor.DocFetcher, drive cvtailor.Drive, editor cvtailor.Editor, creds cvtailor.Credentials) {
	t.Helper()
	gen := cvtailor.NewGenerator(cvtailor.GeneratorDeps{Store: e.store, Docs: docs, Drive: drive, Editor: editor, Creds: creds})
	if err := gen.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick() error = %v", err)
	}
}

func (e draftEnv) bulletResult(text string, cost float64) cvedit.Result {
	return cvedit.Result{
		Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{
			PositionID: e.pos.ID,
			Bullets:    []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: text}},
		}}},
		Cost: cost,
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

func TestGeneratorRunTick(t *testing.T) {
	t.Run("builds a ready Draft from a trimmed copy of the base tab", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		drive := newDrive()
		res := e.bulletResult("Cut p99 latency", 0.5)
		res.Raw = `{"raw":true}`

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, drive, cvtailortest.Editing(res), cvtailortest.Key("sk-or-test"))

		url := "https://docs.google.com/document/d/copy-1/edit"
		wantDraft := dto.Draft{ID: id, JobID: jobID, Status: "ready", DraftDocURL: &url, DraftDocID: "copy-1"}
		if diff := cmp.Diff(wantDraft, e.draft(t, id), cmpopts.IgnoreFields(dto.Draft{}, "Findings")); diff != "" {
			t.Errorf("GetDraft(%s) mismatch (-want +got):\n%s", id, diff)
		}
		wantResult := dto.DraftResult{
			RawOutput: `{"raw":true}`, Model: cvedit.Model, PromptVersion: cvedit.PromptVersion,
			JobFingerprint: "fp-1", Cost: 0.5, DraftDocID: "copy-1",
		}
		if diff := cmp.Diff(wantResult, e.store.DraftResult(id), cmpopts.IgnoreFields(dto.DraftResult{}, "EditSet", "Findings")); diff != "" {
			t.Errorf("recorded result mismatch (-want +got):\n%s", diff)
		}

		if len(drive.Updates) != 2 {
			t.Fatalf("Docs writes = %d batches, want the tab trim then the edits", len(drive.Updates))
		}
		var trim []map[string]map[string]string
		for _, raw := range drive.Updates[0] {
			var req map[string]map[string]string
			if err := json.Unmarshal(raw, &req); err != nil {
				t.Fatalf("decode trim request %s: %v", raw, err)
			}
			trim = append(trim, req)
		}
		wantTrim := []map[string]map[string]string{{"deleteTab": {"tabId": "t.1"}}}
		if diff := cmp.Diff(wantTrim, trim); diff != "" {
			t.Errorf("tab trim mismatch (-want +got):\n%s", diff)
		}
		var inserted []string
		for _, raw := range drive.Updates[1] {
			var req docedit.Request
			if err := json.Unmarshal(raw, &req); err != nil {
				t.Fatalf("decode edit request %s: %v", raw, err)
			}
			if req.InsertText != nil {
				inserted = append(inserted, req.InsertText.Text)
			}
		}
		if diff := cmp.Diff([]string{"Cut p99 latency"}, inserted); diff != "" {
			t.Errorf("inserted text mismatch (-want +got):\n%s", diff)
		}
	})

	refused := []struct {
		name    string
		bullets func(e draftEnv) []cvedit.Bullet
	}{
		{"refuses an unconfirmed achievement", func(e draftEnv) []cvedit.Bullet {
			return []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[1].ID}, Text: "Mentored"}}
		}},
		{"refuses an invented achievement", func(draftEnv) []cvedit.Bullet {
			return []cvedit.Bullet{{AchievementIDs: []string{"nope"}, Text: "Invented"}}
		}},
		{"refuses more bullets than slots", func(e draftEnv) []cvedit.Bullet {
			b := cvedit.Bullet{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: "x"}
			return []cvedit.Bullet{b, b, b, b}
		}},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			e := newDraftEnv(t)
			id := e.queue(t)
			drive := newDrive()
			res := cvedit.Result{Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: e.pos.ID, Bullets: tc.bullets(e)}}}}

			e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, drive, cvtailortest.Editing(res), cvtailortest.Key("sk-or-test"))

			d := e.draft(t, id)
			if d.Status == "ready" || d.LastError == "" {
				t.Errorf("GetDraft(%s) = %+v, want the edit refused with a reason", id, d)
			}
			if len(drive.Copies) != 0 {
				t.Errorf("Drive copies = %v, want none made for a refused edit", drive.Copies)
			}
		})
	}

	t.Run("deletes the copy when a Docs write fails", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		drive := newDrive()
		editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.5))

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, cvtailortest.FailsBatchUpdate(drive, context.DeadlineExceeded), editor, cvtailortest.Key("sk-or-test"))

		if diff := cmp.Diff([]string{"copy-1"}, drive.Deleted); diff != "" {
			t.Errorf("deleted files mismatch (-want +got):\n%s", diff)
		}
		if d := e.draft(t, id); d.Status != "pending" || d.DraftDocURL != nil || d.DraftDocID != "" || d.LastError == "" {
			t.Errorf("GetDraft(%s) = %+v, want a retry pending with a reason and no Doc", id, d)
		}
	})

	t.Run("deletes the copy when the page count export fails", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		drive := newDrive()
		editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.5))

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, cvtailortest.FailsExport(drive, errors.New("export unavailable")), editor, cvtailortest.Key("sk-or-test"))

		if diff := cmp.Diff([]string{"copy-1"}, drive.Deleted); diff != "" {
			t.Errorf("deleted files mismatch (-want +got):\n%s", diff)
		}
		if d := e.draft(t, id); d.Status != "pending" || !strings.Contains(d.LastError, "export unavailable") {
			t.Errorf("GetDraft(%s) = %+v, want a retry pending with the export error", id, d)
		}
	})

	t.Run("deletes the copy when the exported PDF has no pages", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		drive := newDrive()
		editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.5))

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, cvtailortest.ExportsNoPages(drive), editor, cvtailortest.Key("sk-or-test"))

		if diff := cmp.Diff([]string{"copy-1"}, drive.Deleted); diff != "" {
			t.Errorf("deleted files mismatch (-want +got):\n%s", diff)
		}
		if d := e.draft(t, id); d.Status != "pending" || !strings.Contains(d.LastError, "no pages") {
			t.Errorf("GetDraft(%s) = %+v, want a retry pending naming the empty PDF", id, d)
		}
	})

	t.Run("fails for good without an OpenRouter key", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		drive := newDrive()

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, drive, cvtailortest.Editing(cvedit.Result{}), cvtailortest.NoKey{})

		d := e.draft(t, id)
		if d.Status != "failed" || !strings.Contains(d.LastError, "OpenRouter") {
			t.Errorf("GetDraft(%s) = %+v, want failed, naming the missing key", id, d)
		}
		if len(drive.Copies) != 0 {
			t.Errorf("Drive copies = %v, want none", drive.Copies)
		}
	})

	t.Run("retries a blocked edit with its findings", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		blocked, clean := e.bulletResult("Leveraged Postgres", 0.25), e.bulletResult("Moved queries to Postgres", 0.25)
		editor := cvtailortest.Editing(blocked, clean)

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, newDrive(), editor, cvtailortest.Key("sk-or-test"))

		if len(editor.Inputs) != 2 {
			t.Fatalf("editor calls = %d, want 2", len(editor.Inputs))
		}
		retry := editor.Inputs[1]
		if diff := cmp.Diff(&blocked.Edits, retry.PriorEdits); diff != "" {
			t.Errorf("retry PriorEdits mismatch (-want +got):\n%s", diff)
		}
		var priorChecks []string
		for _, f := range retry.PriorFindings {
			priorChecks = append(priorChecks, f.Check)
		}
		if diff := cmp.Diff([]string{"banned_words"}, priorChecks); diff != "" {
			t.Errorf("retry PriorFindings checks mismatch (-want +got):\n%s", diff)
		}
		d := e.draft(t, id)
		if d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
		if got := e.store.DraftResult(id).Cost; got != 0.5 {
			t.Errorf("recorded cost = %v, want 0.5 for both calls", got)
		}
	})

	t.Run("stops after two retries and keeps the findings", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		editor := cvtailortest.Editing(e.bulletResult("Leveraged Postgres", 0.25))

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, newDrive(), editor, cvtailortest.Key("sk-or-test"))

		if len(editor.Inputs) != 3 {
			t.Errorf("editor calls = %d, want 1 attempt and 2 retries", len(editor.Inputs))
		}
		d := e.draft(t, id)
		if diff := cmp.Diff([]string{"banned_words"}, findingChecks(d.Findings, "block")); d.Status != "ready" || diff != "" {
			t.Errorf("GetDraft(%s) = %+v, want ready with the surviving banned-word finding; block checks (-want +got):\n%s", id, d, diff)
		}
	})

	t.Run("keeps the blocked edit when a retry fails", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		editor := cvtailortest.ReplyingWith(
			cvtailortest.Reply{Result: e.bulletResult("Leveraged Postgres", 0.25)},
			cvtailortest.Reply{Result: cvedit.Result{Cost: 0.5}, Err: errors.New("model unavailable")},
		)

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, newDrive(), editor, cvtailortest.Key("sk-or-test"))

		d := e.draft(t, id)
		if diff := cmp.Diff([]string{"banned_words"}, findingChecks(d.Findings, "block")); d.Status != "ready" || diff != "" {
			t.Errorf("GetDraft(%s) = %+v, want ready with the blocked edit's finding kept; block checks (-want +got):\n%s", id, d, diff)
		}
		if got := e.store.DraftResult(id).Cost; got != 0.75 {
			t.Errorf("recorded cost = %v, want 0.75 including the failed call", got)
		}
	})

	t.Run("keeps the blocked edit when a retry returns an invalid edit", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		invalid := cvedit.Result{Cost: 0.5, Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{
			PositionID: e.pos.ID, Bullets: []cvedit.Bullet{{AchievementIDs: []string{"nope"}, Text: "Invented"}},
		}}}}
		editor := cvtailortest.Editing(e.bulletResult("Leveraged Postgres", 0.25), invalid)

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, newDrive(), editor, cvtailortest.Key("sk-or-test"))

		d := e.draft(t, id)
		if diff := cmp.Diff([]string{"banned_words"}, findingChecks(d.Findings, "block")); d.Status != "ready" || diff != "" {
			t.Errorf("GetDraft(%s) = %+v, want ready with the blocked edit's finding kept; block checks (-want +got):\n%s", id, d, diff)
		}
		if got := e.store.DraftResult(id).Cost; got != 0.75 {
			t.Errorf("recorded cost = %v, want 0.75 including the invalid call", got)
		}
	})

	t.Run("shortens once when the draft runs over a page", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		drive := newDrive()
		long := "Cut p99 latency by moving queries"
		editor := cvtailortest.Editing(e.bulletResult(long, 0.25), e.bulletResult("Cut p99 latency", 0.25))

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, cvtailortest.ExportsPages(drive, 1, 2, 1), editor, cvtailortest.Key("sk-or-test"))

		if len(editor.Inputs) != 2 {
			t.Fatalf("editor calls = %d, want 1 attempt and 1 shorten retry", len(editor.Inputs))
		}
		if diff := cmp.Diff([]string{long}, editor.Inputs[1].ShortenBullets); diff != "" {
			t.Errorf("shorten request bullets mismatch (-want +got):\n%s", diff)
		}
		if len(drive.Copies) != 1 {
			t.Errorf("Drive copies = %v, want the same copy edited again", drive.Copies)
		}
		d := e.draft(t, id)
		if d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
	})

	t.Run("records the overflow when shortening still runs over", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.25))

		e.run(t, cvtailortest.Docs{TabJSON: baseTab(t)}, cvtailortest.ExportsPages(newDrive(), 1, 2), editor, cvtailortest.Key("sk-or-test"))

		if len(editor.Inputs) != 2 {
			t.Errorf("editor calls = %d, want 1 attempt and 1 shorten retry", len(editor.Inputs))
		}
		if diff := cmp.Diff([]string{"page_count"}, findingChecks(e.draft(t, id).Findings, "block")); diff != "" {
			t.Errorf("block checks mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("records skill gaps as info findings", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.queue(t)
		tab := tabJSON(t, head(heading), bullet("Built and maintained the public APIs for the platform"), head("Skills"), bullet("Go"), bullet("SQL"))
		res := e.bulletResult("Cut p99 latency", 0.25)
		res.Edits.Skills = []string{"Go", "SQL"}
		res.Edits.JobSkills = []string{"Go", "Kubernetes"}
		editor := cvtailortest.Editing(res)

		e.run(t, cvtailortest.Docs{TabJSON: tab}, newDrive(), editor, cvtailortest.Key("sk-or-test"))

		if !editor.Inputs[0].HasSkills {
			t.Error("editor input HasSkills = false, want the base CV's skills section offered")
		}
		var gaps []dto.DraftFinding
		for _, f := range e.draft(t, id).Findings {
			if f.Severity == "info" {
				gaps = append(gaps, f)
			}
		}
		if len(gaps) != 1 || !strings.Contains(gaps[0].Message, "Kubernetes") {
			t.Errorf("info findings = %+v, want the Kubernetes gap", gaps)
		}
	})
}
