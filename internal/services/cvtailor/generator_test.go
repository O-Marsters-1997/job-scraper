package cvtailor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
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
	docs   cvtailor.DocFetcher
	drive  cvtailor.Drive
	editor cvtailor.Editor
	creds  cvtailor.Credentials
}

func (e draftEnv) run(t *testing.T, tk tick) {
	t.Helper()
	if tk.docs == nil {
		tk.docs = cvtailortest.Docs{TabJSON: baseTab(t)}
	}
	if tk.drive == nil {
		tk.drive = e.drive
	}
	if tk.creds == nil {
		tk.creds = apiKey
	}
	m := cvtailor.Build(cvtailor.Deps{Store: e.store, Docs: tk.docs, Drive: tk.drive, Editor: tk.editor, Creds: tk.creds})
	if err := m.RunTick(t.Context()); err != nil {
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
		e, id := newQueuedDraft(t)
		res := e.bulletResult("Cut p99 latency", 0.5)
		res.Raw = `{"raw":true}`

		e.run(t, tick{editor: cvtailortest.Editing(res)})

		url := "https://docs.google.com/document/d/copy-1/edit"
		wantDraft := dto.Draft{ID: id, JobID: jobID, Status: "ready", DraftDocURL: &url, DraftDocID: "copy-1"}
		if diff := cmp.Diff(wantDraft, e.draft(t, id), cmpopts.IgnoreFields(dto.Draft{}, "Findings", "CreatedAt", "EditSet", "Provenance", "Content", "Base", "BaseContent")); diff != "" {
			t.Errorf("GetDraft(%s) mismatch (-want +got):\n%s", id, diff)
		}
		wantResult := dto.DraftResult{
			RawOutput: `{"raw":true}`, Model: cvedit.Model, PromptVersion: cvedit.PromptVersion,
			JobFingerprint: "fp-1", Cost: 0.5, DraftDocID: "copy-1",
		}
		if diff := cmp.Diff(wantResult, e.store.DraftResult(id), cmpopts.IgnoreFields(dto.DraftResult{}, "EditSet", "BaseContent", "Findings")); diff != "" {
			t.Errorf("recorded result mismatch (-want +got):\n%s", diff)
		}

		if len(e.drive.Updates) != 2 {
			t.Fatalf("Docs writes = %d batches, want the tab trim then the edits", len(e.drive.Updates))
		}
		var trim []map[string]map[string]string
		for _, raw := range e.drive.Updates[0] {
			trim = append(trim, handlerstest.DecodeJSON[map[string]map[string]string](t, raw))
		}
		wantTrim := []map[string]map[string]string{{"deleteTab": {"tabId": "t.1"}}}
		if diff := cmp.Diff(wantTrim, trim); diff != "" {
			t.Errorf("tab trim mismatch (-want +got):\n%s", diff)
		}
		var inserted []string
		for _, raw := range e.drive.Updates[1] {
			if req := handlerstest.DecodeJSON[docedit.Request](t, raw); req.InsertText != nil {
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
			e, id := newQueuedDraft(t)
			res := cvedit.Result{Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: e.pos.ID, Bullets: tc.bullets(e)}}}}

			e.run(t, tick{editor: cvtailortest.Editing(res)})

			d := e.draft(t, id)
			if d.Status == "ready" || d.LastError == "" {
				t.Errorf("GetDraft(%s) = %+v, want the edit refused with a reason", id, d)
			}
			if len(e.drive.Copies) != 0 {
				t.Errorf("Drive copies = %v, want none made for a refused edit", e.drive.Copies)
			}
		})
	}

	driveFailures := []struct {
		name    string
		wrap    func(cvtailor.Drive) cvtailor.Drive
		wantErr string
	}{
		{"a Docs write fails", func(d cvtailor.Drive) cvtailor.Drive {
			return cvtailortest.FailsBatchUpdate(d, context.DeadlineExceeded)
		}, context.DeadlineExceeded.Error()},
		{"the page count export fails", func(d cvtailor.Drive) cvtailor.Drive {
			return cvtailortest.FailsExport(d, errors.New("export unavailable"))
		}, "export unavailable"},
		{"the exported PDF has no pages", cvtailortest.ExportsNoPages, "no pages"},
	}
	for _, tc := range driveFailures {
		t.Run("deletes the copy when "+tc.name, func(t *testing.T) {
			e, id := newQueuedDraft(t)
			editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.5))

			e.run(t, tick{drive: tc.wrap(e.drive), editor: editor})

			if diff := cmp.Diff([]string{"copy-1"}, e.drive.Deleted); diff != "" {
				t.Errorf("deleted files mismatch (-want +got):\n%s", diff)
			}
			d := e.draft(t, id)
			if d.Status != "pending" || d.DraftDocURL != nil || d.DraftDocID != "" || !strings.Contains(d.LastError, tc.wantErr) {
				t.Errorf("GetDraft(%s) = %+v, want a retry pending with %q and no Doc", id, d, tc.wantErr)
			}
		})
	}

	t.Run("fails for good without an OpenRouter key", func(t *testing.T) {
		e, id := newQueuedDraft(t)

		e.run(t, tick{editor: cvtailortest.Editing(cvedit.Result{}), creds: cvtailortest.NoKey{}})

		d := e.draft(t, id)
		if d.Status != "failed" || !strings.Contains(d.LastError, "OpenRouter") {
			t.Errorf("GetDraft(%s) = %+v, want failed, naming the missing key", id, d)
		}
		if len(e.drive.Copies) != 0 {
			t.Errorf("Drive copies = %v, want none", e.drive.Copies)
		}
	})

	t.Run("leaves a kept slot untouched", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		kept := cvedit.Result{Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{
			PositionID: e.pos.ID,
			Bullets:    []cvedit.Bullet{{Keep: true, Text: "Built and maintained the public APIs for the platform"}},
		}}}}

		e.run(t, tick{editor: cvtailortest.Editing(kept)})

		if d := e.draft(t, id); d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
		for _, raw := range e.drive.Updates[1] {
			if req := handlerstest.DecodeJSON[docedit.Request](t, raw); req.InsertText != nil {
				t.Errorf("inserted %q, want the kept slot left as it is", req.InsertText.Text)
			}
		}
	})

	t.Run("puts a blocked bullet back to its original text without another call", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		editor := cvtailortest.Editing(e.bulletResult("Leveraged Postgres", 0.25))

		e.run(t, tick{editor: editor})

		if len(editor.Inputs) != 1 {
			t.Errorf("editor calls = %d, want 1", len(editor.Inputs))
		}
		d := e.draft(t, id)
		if d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
		for _, raw := range e.drive.Updates[1] {
			if req := handlerstest.DecodeJSON[docedit.Request](t, raw); req.InsertText != nil {
				t.Errorf("inserted %q, want the blocked slot left as it was", req.InsertText.Text)
			}
		}
		if got := e.store.DraftResult(id).Cost; got != 0.25 {
			t.Errorf("recorded cost = %v, want 0.25", got)
		}
	})

	t.Run("drops a skills reorder that names an unsupported skill", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		res := e.bulletResult("Cut p99 latency", 0.25)
		res.Edits.Skills = []string{"Rust", "Go"}

		e.run(t, tick{docs: cvtailortest.Docs{TabJSON: baseTab(t, head("Skills"), bullet("Go"), bullet("SQL"))}, editor: cvtailortest.Editing(res)})

		if d := e.draft(t, id); d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
		for _, raw := range e.drive.Updates[1] {
			if req := handlerstest.DecodeJSON[docedit.Request](t, raw); req.InsertText != nil && strings.Contains(req.InsertText.Text, "Rust") {
				t.Errorf("inserted %q, want the original skills kept", req.InsertText.Text)
			}
		}
	})

	t.Run("shortens once when the draft runs over a page", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		long := "Cut p99 latency by moving queries"
		editor := cvtailortest.Editing(e.bulletResult(long, 0.25), e.bulletResult("Cut p99 latency", 0.25))

		e.run(t, tick{drive: cvtailortest.ExportsPages(e.drive, 1, 2, 1), editor: editor})

		if len(editor.Inputs) != 2 {
			t.Fatalf("editor calls = %d, want 1 attempt and 1 shorten retry", len(editor.Inputs))
		}
		if diff := cmp.Diff([]string{long}, editor.Inputs[1].ShortenBullets); diff != "" {
			t.Errorf("shorten request bullets mismatch (-want +got):\n%s", diff)
		}
		if len(e.drive.Copies) != 1 {
			t.Errorf("Drive copies = %v, want the same copy edited again", e.drive.Copies)
		}
		d := e.draft(t, id)
		if d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
	})

	t.Run("records the overflow when shortening still runs over", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.25))

		e.run(t, tick{drive: cvtailortest.ExportsPages(e.drive, 1, 2), editor: editor})

		if len(editor.Inputs) != 2 {
			t.Errorf("editor calls = %d, want 1 attempt and 1 shorten retry", len(editor.Inputs))
		}
		if diff := cmp.Diff([]string{"page_count"}, findingChecks(e.draft(t, id).Findings, "block")); diff != "" {
			t.Errorf("block checks mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("plans the shorten retry against the original CV", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		docs := cvtailortest.EditedCopyDocs{
			BaseDocID: docID,
			Base:      baseTab(t, head("Skills"), bullet("Go"), bullet("SQL")),
			Copy: tabJSON(t, head(heading), bullet("Cut p99 latency by moving queries"), bullet("Ran on-call"), bullet("Wrote docs"),
				head("Skills"), bullet("Go"), bullet("SQL"), bullet("Kubernetes")),
		}
		editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency by moving queries", 0.25), e.bulletResult("Cut p99 latency", 0.25))

		e.run(t, tick{docs: docs, drive: cvtailortest.ExportsPages(e.drive, 1, 2, 1), editor: editor})

		if len(editor.Inputs) != 2 {
			t.Fatalf("editor calls = %d, want 1 attempt and 1 shorten retry", len(editor.Inputs))
		}
		first, shorten := editor.Inputs[0], editor.Inputs[1]
		if diff := cmp.Diff(first.BaseSkills, shorten.BaseSkills); diff != "" {
			t.Errorf("shorten BaseSkills mismatch (-original +got):\n%s", diff)
		}
		if diff := cmp.Diff(first.Positions, shorten.Positions); diff != "" {
			t.Errorf("shorten Positions mismatch (-original +got):\n%s", diff)
		}
		if d := e.draft(t, id); d.Status != "ready" {
			t.Errorf("GetDraft(%s) = %+v, want ready", id, d)
		}
	})

	t.Run("puts a blocked shortened bullet back without another call", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		editor := cvtailortest.Editing(
			e.bulletResult("Cut p99 latency by moving queries", 0.25),
			e.bulletResult("Leveraged Postgres", 0.25),
		)

		e.run(t, tick{drive: cvtailortest.ExportsPages(e.drive, 1, 2, 1), editor: editor})

		if len(editor.Inputs) != 2 {
			t.Fatalf("editor calls = %d, want 1 attempt and 1 shorten", len(editor.Inputs))
		}
		d := e.draft(t, id)
		if d.Status != "ready" || len(findingChecks(d.Findings, "block")) != 0 {
			t.Errorf("GetDraft(%s) = %+v, want ready with no block findings", id, d)
		}
	})

	t.Run("records skill gaps as info findings", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		res := e.bulletResult("Cut p99 latency", 0.25)
		res.Edits.Skills = []string{"Go", "SQL"}
		res.Edits.JobSkills = []string{"Go", "Kubernetes"}
		editor := cvtailortest.Editing(res)

		e.run(t, tick{docs: cvtailortest.Docs{TabJSON: baseTab(t, head("Skills"), bullet("Go"), bullet("SQL"))}, editor: editor})

		if !editor.Inputs[0].HasSkills {
			t.Error("editor input HasSkills = false, want the base CV's skills section offered")
		}
		var gaps []dto.DraftFinding
		for _, f := range e.draft(t, id).Findings {
			if f.Severity == "info" && f.Check == "skills" {
				gaps = append(gaps, f)
			}
		}
		if len(gaps) != 1 || !strings.Contains(gaps[0].Message, "Kubernetes") {
			t.Errorf("info findings = %+v, want the Kubernetes gap", gaps)
		}
	})

	t.Run("warns when contact details are only in the Doc header", func(t *testing.T) {
		e, id := newQueuedDraft(t)
		tab := bytes.Replace(baseTab(t), []byte(`{"documentTab":{`),
			[]byte(`{"documentTab":{"headers":{"h":{"content":[{"paragraph":{"elements":[{"textRun":{"content":"jo@example.com\n"}}]}}]}},`), 1)

		e.run(t, tick{docs: cvtailortest.Docs{TabJSON: tab}, editor: cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.25))})

		var got []string
		for _, f := range e.draft(t, id).Findings {
			if f.Check == "contact" {
				got = append(got, f.Severity)
			}
		}
		if diff := cmp.Diff([]string{"info"}, got); diff != "" {
			t.Errorf("contact findings mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("flags a layout the PDF text extractor reads badly", func(t *testing.T) {
		tests := []struct {
			name, file string
			want       []string
		}{
			{"single column draft has no parse findings", "draft-single-column.pdf", nil},
			{"heading sharing a row is flagged", "draft-table.pdf", []string{"parse"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pdf, err := os.ReadFile("testdata/" + tt.file)
				if err != nil {
					t.Fatal(err)
				}
				e, id := newQueuedDraft(t)
				editor := cvtailortest.Editing(e.bulletResult("Cut p99 latency", 0.25))

				e.run(t, tick{drive: cvtailortest.ExportsPDF(e.drive, string(pdf)), editor: editor})

				var got []string
				for _, c := range findingChecks(e.draft(t, id).Findings, "info") {
					if c == "parse" {
						got = append(got, c)
					}
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("parse findings mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}
