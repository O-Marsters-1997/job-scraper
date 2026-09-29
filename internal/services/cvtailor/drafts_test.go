package cvtailor_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type editorFunc func(cvedit.Input) (cvedit.Result, error)

func (f editorFunc) Edit(_ context.Context, _ string, in cvedit.Input) (cvedit.Result, error) {
	return f(in)
}

type credentials struct{ err error }

func (c credentials) Get(context.Context, string, string) (string, error) { return "sk-or-test", c.err }

const (
	user    = handlerstest.UserID
	jobID   = "job-1"
	docID   = "doc-1"
	tabID   = "t.0"
	heading = "Engineer, Acme"
)

type draftEnv struct {
	store *cvtailortest.FakeStore
	drive *cvtailortest.Drive
	svc   *cvtailor.Service
	pos   dto.Position
	input dto.DraftInput
}

func newDraftEnv(t *testing.T) draftEnv {
	t.Helper()
	store := cvtailortest.NewFakeStore()
	ctx := context.Background()
	pos, err := store.CreatePosition(ctx, user, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Cut p99 latency by moving queries to Postgres", "Mentored four engineers"} {
		a, err := store.CreateAchievement(ctx, user, dto.AchievementInput{PositionID: pos.ID, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		pos.Achievements = append(pos.Achievements, a)
	}
	if err := store.SaveHeadingMappings(ctx, user, docID, tabID, []dto.HeadingMapping{{HeadingText: heading, PositionID: &pos.ID}}); err != nil {
		t.Fatal(err)
	}
	store.SetJob(jobID, "We need a Go engineer.", "fp-1")
	drive := &cvtailortest.Drive{Tabs: []google.Tab{{ID: tabID}, {ID: "t.1"}}}
	return draftEnv{
		store: store,
		drive: drive,
		svc:   cvtailor.NewService(store, nil, nil, drive),
		pos:   pos,
		input: dto.DraftInput{JobID: jobID, DocID: docID, TabID: tabID, AchievementIDs: []string{pos.Achievements[0].ID}},
	}
}

func (e draftEnv) generator(t *testing.T, editor editorFunc, creds credentials) *cvtailor.Generator {
	t.Helper()
	return e.generatorFor(t, tabJSON(t, head(heading), bullet("Built and maintained the public APIs for the platform"), bullet("Ran on-call"), bullet("Wrote docs")), editor, creds)
}

func (e draftEnv) generatorFor(t *testing.T, tab json.RawMessage, editor editorFunc, creds credentials) *cvtailor.Generator {
	t.Helper()
	docs := cvtailortest.Docs{TabJSON: tab}
	return cvtailor.NewGenerator(cvtailor.GeneratorDeps{
		Store: e.store, Docs: docs, Drive: e.drive, Editor: editor, Creds: creds,
	})
}

func (e draftEnv) queue(t *testing.T) string {
	t.Helper()
	ref, err := e.svc.CreateDraft(context.Background(), user, e.input)
	if err != nil {
		t.Fatalf("CreateDraft() err = %v", err)
	}
	return ref.ID
}

func (e draftEnv) draft(t *testing.T, id string) dto.Draft {
	t.Helper()
	d, err := e.svc.GetDraft(context.Background(), user, dto.DraftQuery{ID: id})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func editing(bullets ...cvedit.Bullet) editorFunc {
	return func(in cvedit.Input) (cvedit.Result, error) {
		return cvedit.Result{
			Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: in.Positions[0].ID, Bullets: bullets}}},
			Cost:  0.02, Raw: `{"raw":true}`,
		}, nil
	}
}

func TestGeneratorBuildsReadyDraftFromTrimmedCopy(t *testing.T) {
	e := newDraftEnv(t)
	id := e.queue(t)
	cited := e.pos.Achievements[0].ID

	err := e.generator(t, editing(cvedit.Bullet{AchievementIDs: []string{cited}, Text: "Cut p99 latency"}), credentials{}).RunTick(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	d := e.draft(t, id)
	if d.Status != "ready" || d.DraftDocURL == nil || !strings.Contains(*d.DraftDocURL, "copy-1") {
		t.Fatalf("GetDraft() = %+v, want ready and linked to the copy", d)
	}
	if len(e.drive.Updates) != 2 || !strings.Contains(string(e.drive.Updates[0][0]), `"deleteTab":{"tabId":"t.1"}`) {
		t.Fatalf("updates = %s, want the other tab removed first", e.drive.Updates)
	}
	edits := string(mustJSON(t, e.drive.Updates[1]))
	if !strings.Contains(edits, "Cut p99 latency") || strings.Count(edits, "deleteContentRange") < 3 {
		t.Errorf("edit requests = %s, want the new bullet and the two unused slots deleted", edits)
	}
	res := e.store.DraftResult(id)
	if res.Model != cvedit.Model || res.PromptVersion != cvedit.PromptVersion || res.JobFingerprint != "fp-1" || res.Cost != 0.02 || res.RawOutput != `{"raw":true}` {
		t.Errorf("recorded result = %+v, want model, prompt version, fingerprint, cost and raw output", res)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGeneratorRejectsEditsOutsideTheConfirmedSet(t *testing.T) {
	cases := []struct {
		name    string
		bullets func(e draftEnv) []cvedit.Bullet
	}{
		{"unconfirmed achievement", func(e draftEnv) []cvedit.Bullet {
			return []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[1].ID}, Text: "Mentored"}}
		}},
		{"invented achievement", func(draftEnv) []cvedit.Bullet {
			return []cvedit.Bullet{{AchievementIDs: []string{"nope"}, Text: "Invented"}}
		}},
		{"more bullets than slots", func(e draftEnv) []cvedit.Bullet {
			b := cvedit.Bullet{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: "x"}
			return []cvedit.Bullet{b, b, b, b}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newDraftEnv(t)
			id := e.queue(t)
			if err := e.generator(t, editing(tc.bullets(e)...), credentials{}).RunTick(context.Background()); err != nil {
				t.Fatal(err)
			}
			d := e.draft(t, id)
			if d.Status == "ready" || d.LastError == "" {
				t.Errorf("GetDraft() = %+v, want the edit refused with a reason", d)
			}
			if len(e.drive.Copies) != 0 {
				t.Errorf("copies = %v, want none made for a refused edit", e.drive.Copies)
			}
		})
	}
}

func TestGeneratorDeletesTheCopyWhenAFailureLeavesItBehind(t *testing.T) {
	e := newDraftEnv(t)
	e.drive.BatchUpdateErr = context.DeadlineExceeded
	id := e.queue(t)
	gen := e.generator(t, editing(cvedit.Bullet{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: "Cut p99 latency"}), credentials{})

	if err := gen.RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(e.drive.Copies) != 1 || len(e.drive.Deleted) != 1 || e.drive.Deleted[0] != e.drive.Copies[0] {
		t.Errorf("copies = %v, deleted = %v, want the copy deleted", e.drive.Copies, e.drive.Deleted)
	}
	if d := e.draft(t, id); d.Status != "pending" || d.DraftDocURL != nil || d.LastError == "" {
		t.Errorf("GetDraft() = %+v, want a retry pending with a reason and no Doc", d)
	}
}

func TestGeneratorFailsForGoodWithoutAnOpenRouterKey(t *testing.T) {
	e := newDraftEnv(t)
	id := e.queue(t)

	if err := e.generator(t, editing(), credentials{err: data.ErrNotFound}).RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	d := e.draft(t, id)
	if d.Status != "failed" || !strings.Contains(d.LastError, "OpenRouter") {
		t.Errorf("GetDraft() = %+v, want failed, naming the missing key", d)
	}
	if len(e.drive.Copies) != 0 {
		t.Errorf("copies = %v, want none", e.drive.Copies)
	}
}

func TestCreateDraftValidation(t *testing.T) {
	e := newDraftEnv(t)
	other, err := e.store.CreatePosition(context.Background(), user, dto.PositionInput{Employer: "Unmapped", Title: "Dev"})
	if err != nil {
		t.Fatal(err)
	}
	unmapped, err := e.store.CreateAchievement(context.Background(), user, dto.AchievementInput{PositionID: other.ID, Text: "Shipped"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		user   string
		mutate func(*dto.DraftInput)
	}{
		{"no achievements", user, func(in *dto.DraftInput) { in.AchievementIDs = nil }},
		{"repeated achievement", user, func(in *dto.DraftInput) { in.AchievementIDs = []string{in.AchievementIDs[0], in.AchievementIDs[0]} }},
		{"another user's achievement", "user-2", func(*dto.DraftInput) {}},
		{"achievement of an unmapped position", user, func(in *dto.DraftInput) { in.AchievementIDs = []string{unmapped.ID} }},
		{"missing tab", user, func(in *dto.DraftInput) { in.TabID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := e.input
			in.AchievementIDs = append([]string(nil), in.AchievementIDs...)
			tc.mutate(&in)
			_, err := e.svc.CreateDraft(context.Background(), tc.user, in)
			if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
				t.Errorf("CreateDraft() err = %v, want an invalid error", err)
			}
		})
	}
}

type recordingEditor struct {
	inputs  []cvedit.Input
	replies []cvedit.Result
}

func (r *recordingEditor) edit(in cvedit.Input) (cvedit.Result, error) {
	r.inputs = append(r.inputs, in)
	return r.replies[min(len(r.inputs), len(r.replies))-1], nil
}

func bulletResult(e draftEnv, text string) cvedit.Result {
	return cvedit.Result{
		Edits: cvedit.EditSet{Positions: []cvedit.PositionEdit{{
			PositionID: e.pos.ID,
			Bullets:    []cvedit.Bullet{{AchievementIDs: []string{e.pos.Achievements[0].ID}, Text: text}},
		}}},
		Cost: 0.01,
	}
}

func blockFindings(d dto.Draft) []dto.DraftFinding {
	var out []dto.DraftFinding
	for _, f := range d.Findings {
		if f.Severity == "block" {
			out = append(out, f)
		}
	}
	return out
}

func TestGeneratorRetriesABlockedEditWithItsFindings(t *testing.T) {
	e := newDraftEnv(t)
	id := e.queue(t)
	ed := &recordingEditor{replies: []cvedit.Result{
		bulletResult(e, "Leveraged Postgres"),
		bulletResult(e, "Moved queries to Postgres"),
	}}

	if err := e.generator(t, ed.edit, credentials{}).RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(ed.inputs) != 2 {
		t.Fatalf("editor calls = %d, want 2", len(ed.inputs))
	}
	retry := ed.inputs[1]
	if retry.PriorEdits == nil || len(retry.PriorFindings) != 1 || retry.PriorFindings[0].Check != "banned_words" {
		t.Errorf("retry input = %+v, want the prior edits and the banned-word finding", retry)
	}
	d := e.draft(t, id)
	if d.Status != "ready" || len(blockFindings(d)) != 0 {
		t.Errorf("GetDraft() = %+v, want ready with no block findings", d)
	}
	if got := e.store.DraftResult(id).Cost; got != 0.02 {
		t.Errorf("recorded cost = %v, want both calls billed", got)
	}
}

func TestGeneratorStopsAfterTwoRetriesAndKeepsTheFindings(t *testing.T) {
	e := newDraftEnv(t)
	id := e.queue(t)
	ed := &recordingEditor{replies: []cvedit.Result{bulletResult(e, "Leveraged Postgres")}}

	if err := e.generator(t, ed.edit, credentials{}).RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(ed.inputs) != 3 {
		t.Errorf("editor calls = %d, want 1 attempt and 2 retries", len(ed.inputs))
	}
	d := e.draft(t, id)
	blocks := blockFindings(d)
	if d.Status != "ready" || len(blocks) != 1 || blocks[0].Check != "banned_words" {
		t.Errorf("GetDraft() = %+v, want ready with the surviving banned-word finding", d)
	}
}

func TestGeneratorShortensOnceWhenTheDraftRunsOverAPage(t *testing.T) {
	e := newDraftEnv(t)
	e.drive.DraftPages = []int{2, 1}
	id := e.queue(t)
	ed := &recordingEditor{replies: []cvedit.Result{
		bulletResult(e, "Cut p99 latency by moving queries"),
		bulletResult(e, "Cut p99 latency"),
	}}

	if err := e.generator(t, ed.edit, credentials{}).RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(ed.inputs) != 2 || !slices.Contains(ed.inputs[1].ShortenBullets, "Cut p99 latency by moving queries") {
		t.Fatalf("editor inputs = %+v, want one shorten retry naming the long bullet", ed.inputs)
	}
	if len(e.drive.Copies) != 1 || len(e.drive.Updates) != 3 {
		t.Errorf("copies = %v, updates = %d, want the same copy edited again", e.drive.Copies, len(e.drive.Updates))
	}
	if d := e.draft(t, id); d.Status != "ready" || len(blockFindings(d)) != 0 {
		t.Errorf("GetDraft() = %+v, want ready with no block findings", d)
	}
}

func TestGeneratorShortensOnlyOnceAndRecordsTheOverflow(t *testing.T) {
	e := newDraftEnv(t)
	e.drive.DraftPages = []int{2}
	id := e.queue(t)
	ed := &recordingEditor{replies: []cvedit.Result{bulletResult(e, "Cut p99 latency")}}

	if err := e.generator(t, ed.edit, credentials{}).RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(ed.inputs) != 2 {
		t.Errorf("editor calls = %d, want 1 attempt and 1 shorten retry", len(ed.inputs))
	}
	blocks := blockFindings(e.draft(t, id))
	if len(blocks) != 1 || blocks[0].Check != "page_count" {
		t.Errorf("block findings = %+v, want the page_count overflow", blocks)
	}
}

func TestGeneratorRecordsSkillGapsAsInfoFindings(t *testing.T) {
	e := newDraftEnv(t)
	id := e.queue(t)
	tab := tabJSON(t, head(heading), bullet("Built and maintained the public APIs for the platform"), head("Skills"), bullet("Go"), bullet("SQL"))
	res := bulletResult(e, "Cut p99 latency")
	res.Edits.Skills = []string{"Go", "SQL"}
	res.Edits.JobSkills = []string{"Go", "Kubernetes"}
	ed := &recordingEditor{replies: []cvedit.Result{res}}

	if err := e.generatorFor(t, tab, ed.edit, credentials{}).RunTick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !ed.inputs[0].HasSkills {
		t.Error("editor input HasSkills = false, want the base CV's skills section offered")
	}
	var infos []dto.DraftFinding
	for _, f := range e.draft(t, id).Findings {
		if f.Severity == "info" {
			infos = append(infos, f)
		}
	}
	if len(infos) != 1 || !strings.Contains(infos[0].Message, "Kubernetes") {
		t.Errorf("info findings = %+v, want the Kubernetes gap", infos)
	}
}
