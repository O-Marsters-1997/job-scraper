package cvtailortest

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func runDraftContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	input := func(f Fixture) dto.DraftInput {
		return dto.DraftInput{JobID: f.JobID, DocID: "doc", TabID: "t.0", AchievementIDs: []string{missingID}}
	}
	create := func(t *testing.T, f Fixture) dto.Draft {
		t.Helper()
		d, err := f.Store.CreateDraft(ctx, f.UserID, input(f), nil)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	claim := func(t *testing.T, f Fixture) dto.DraftClaim {
		t.Helper()
		c, err := f.Store.ClaimDraft(ctx)
		if err != nil {
			t.Fatalf("ClaimDraft() err = %v", err)
		}
		return c
	}

	t.Run("a created Draft is pending and private to its owner", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		if d.Status != "pending" {
			t.Errorf("CreateDraft().Status = %q, want pending", d.Status)
		}
		got, err := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if err != nil || got.ID != d.ID {
			t.Fatalf("GetDraft() = %+v, %v", got, err)
		}
		if _, err := f.Store.GetDraft(ctx, f.Other, d.ID); err == nil {
			t.Error("GetDraft(other user) err = nil, want not found")
		}
	})

	t.Run("a Draft is claimed once and carries its request", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		c := claim(t, f)
		if c.ID != d.ID || c.UserID != f.UserID || c.DocID != "doc" || c.TabID != "t.0" || len(c.AchievementIDs) != 1 || c.Attempts != 1 {
			t.Errorf("ClaimDraft() = %+v, want the created Draft on attempt 1", c)
		}
		if _, err := f.Store.ClaimDraft(ctx); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("second ClaimDraft() err = %v, want ErrNotFound", err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "running" {
			t.Errorf("GetDraft().Status = %q, want running", got.Status)
		}
	})

	t.Run("a claim carries the Draft's skill swaps", func(t *testing.T) {
		f := newStore(t)
		in := input(f)
		in.SkillSwaps = []dto.SkillSwap{{BankSkillID: "b1", Line: 1, Replaces: "SQL"}}
		if _, err := f.Store.CreateDraft(ctx, f.UserID, in, nil); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(in.SkillSwaps, claim(t, f).SkillSwaps); diff != "" {
			t.Errorf("ClaimDraft().SkillSwaps mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("completing records the result and the Doc", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		c := claim(t, f)
		finding := dto.DraftFinding{Check: "skills", Severity: "info", Message: "the job asks for Rust"}
		res := dto.DraftResult{EditSet: json.RawMessage(`{"positions":[]}`), BaseContent: json.RawMessage(`{"profile":"hi"}`), Model: "m", PromptVersion: "p", DraftDocID: "doc-copy", Cost: 0.5, Findings: []dto.DraftFinding{finding}}
		if err := f.Store.CompleteDraft(ctx, c, res); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "ready" || got.DraftDocID != "doc-copy" {
			t.Errorf("GetDraft() = %+v, want ready with the Doc", got)
		}
		var base dto.DraftContent
		if err := json.Unmarshal(got.BaseContent, &base); err != nil || base.Profile == nil || *base.Profile != "hi" {
			t.Errorf("GetDraft().BaseContent = %s, %v, want the recorded base content", got.BaseContent, err)
		}
		if diff := cmp.Diff([]dto.DraftFinding{finding}, got.Findings); diff != "" {
			t.Errorf("GetDraft().Findings mismatch (-want +got):\n%s", diff)
		}
		if err := f.Store.CompleteDraft(ctx, c, res); err == nil {
			t.Error("second CompleteDraft() err = nil, want a stale-claim error")
		}
	})

	t.Run("saving edits replaces the edit set and findings for the owner only", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		if err := f.Store.CompleteDraft(ctx, claim(t, f), dto.DraftResult{EditSet: json.RawMessage(`{"positions":[]}`), DraftDocID: "doc-copy"}); err != nil {
			t.Fatal(err)
		}
		finding := dto.DraftFinding{Check: "page_count", Severity: "block", Message: "over a page"}
		edited := json.RawMessage(`{"positions":[{"positionId":"p","bullets":[]}]}`)
		if err := f.Store.SetDraftEdits(ctx, f.Other, d.ID, edited, nil); !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("SetDraftEdits(other user) err = %v, want not found", err)
		}
		if err := f.Store.SetDraftEdits(ctx, f.UserID, d.ID, edited, []dto.DraftFinding{finding}); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if diff := cmp.Diff([]dto.DraftFinding{finding}, got.Findings); diff != "" {
			t.Errorf("GetDraft().Findings mismatch (-want +got):\n%s", diff)
		}
		var gotEdits, wantEdits any
		if err := json.Unmarshal(got.EditSet, &gotEdits); err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(edited, &wantEdits)
		if diff := cmp.Diff(wantEdits, gotEdits); diff != "" {
			t.Errorf("GetDraft().EditSet mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("a terminal failure is failed with its reason and no Doc", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		c := claim(t, f)
		if err := f.Store.SetDraftDoc(ctx, c, "orphan"); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.FailDraft(ctx, c, dto.DraftFailure{Reason: "no key", Terminal: true, ClearDoc: true}); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "failed" || got.LastError != "no key" || got.DraftDocID != "" {
			t.Errorf("GetDraft() = %+v, want failed, last_error set, no Doc", got)
		}
	})

	t.Run("a retryable failure backs off instead of being re-claimed at once", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		c := claim(t, f)
		if err := f.Store.FailDraft(ctx, c, dto.DraftFailure{Reason: "flaky"}); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "pending" {
			t.Errorf("GetDraft().Status = %q, want pending", got.Status)
		}
		if _, err := f.Store.ClaimDraft(ctx); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("ClaimDraft() err = %v, want ErrNotFound while backing off", err)
		}
	})

	t.Run("a Job's Drafts list newest first and only for their owner", func(t *testing.T) {
		f := newStore(t)
		first, second := create(t, f), create(t, f)
		got, err := f.Store.ListJobDrafts(ctx, f.UserID, f.JobID)
		if err != nil || len(got) != 2 {
			t.Fatalf("ListJobDrafts() = %+v, %v, want two Drafts", got, err)
		}
		if got[0].ID != second.ID || got[1].ID != first.ID {
			t.Errorf("ListJobDrafts() ids = %s, %s, want %s, %s", got[0].ID, got[1].ID, second.ID, first.ID)
		}
		other, err := f.Store.ListJobDrafts(ctx, f.Other, f.JobID)
		if err != nil || len(other) != 0 {
			t.Errorf("ListJobDrafts(other user) = %+v, %v, want none", other, err)
		}
	})

	t.Run("a Job keeps at most one Draft", func(t *testing.T) {
		f := newStore(t)
		first, second := create(t, f), create(t, f)
		kept, err := f.Store.SetDraftOutcome(ctx, f.UserID, first.ID, dto.OutcomeKept)
		if err != nil || kept.Outcome == nil || *kept.Outcome != dto.OutcomeKept {
			t.Fatalf("SetDraftOutcome(kept) = %+v, %v", kept, err)
		}
		_, err = f.Store.SetDraftOutcome(ctx, f.UserID, second.ID, dto.OutcomeKept)
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindConflict.Status() {
			t.Errorf("second SetDraftOutcome(kept) err = %v, want a conflict", err)
		}
		if _, err := f.Store.SetDraftOutcome(ctx, f.UserID, first.ID, dto.OutcomeDiscarded); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Store.SetDraftOutcome(ctx, f.UserID, second.ID, dto.OutcomeKept); err != nil {
			t.Errorf("SetDraftOutcome(kept) after discarding the first err = %v, want nil", err)
		}
	})

	t.Run("discarding drops the Doc id and keeps the row", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		c := claim(t, f)
		if err := f.Store.CompleteDraft(ctx, c, dto.DraftResult{EditSet: json.RawMessage(`{}`), DraftDocID: "doc-copy"}); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Store.SetDraftOutcome(ctx, f.UserID, d.ID, dto.OutcomeDiscarded); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if err != nil || got.DraftDocID != "" || got.Outcome == nil || *got.Outcome != dto.OutcomeDiscarded {
			t.Errorf("GetDraft() = %+v, %v, want discarded with no Doc", got, err)
		}
		if _, err := f.Store.SetDraftOutcome(ctx, f.Other, d.ID, dto.OutcomeKept); err == nil {
			t.Error("SetDraftOutcome(other user) err = nil, want not found")
		}
	})

	ready := func(t *testing.T, f Fixture) dto.Draft {
		t.Helper()
		d := create(t, f)
		c := claim(t, f)
		if err := f.Store.CompleteDraft(ctx, c, dto.DraftResult{EditSet: json.RawMessage(`{}`), DraftDocID: "doc-copy"}); err != nil {
			t.Fatal(err)
		}
		return d
	}

	t.Run("keeping a ready Draft queues it and the claim completes the keep", func(t *testing.T) {
		f := newStore(t)
		d := ready(t, f)
		queued, err := f.Store.QueueKeep(ctx, f.UserID, d.ID)
		if err != nil || queued.Status != "keeping" || queued.Outcome != nil {
			t.Fatalf("QueueKeep() = %+v, %v, want keeping and undecided", queued, err)
		}
		if _, err := f.Store.QueueKeep(ctx, f.Other, d.ID); err == nil {
			t.Error("QueueKeep(other user) err = nil, want not found")
		}
		c := claim(t, f)
		if !c.Keeping || c.ID != d.ID || c.DraftDocID != "doc-copy" || c.Attempts != 1 {
			t.Fatalf("ClaimDraft() = %+v, want the keeping Draft on attempt 1", c)
		}
		if _, err := f.Store.ClaimDraft(ctx); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("ClaimDraft() while leased err = %v, want ErrNotFound", err)
		}
		if err := f.Store.CompleteKeep(ctx, c); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "ready" || got.Outcome == nil || *got.Outcome != dto.OutcomeKept || got.KeptAs != "doc" {
			t.Errorf("GetDraft() = %+v, want ready, kept as a doc", got)
		}
		if _, err := f.Store.QueueKeep(ctx, f.UserID, d.ID); err == nil {
			t.Error("QueueKeep(kept) err = nil, want not found")
		}
	})

	t.Run("a failed keep backs off instead of being re-claimed at once", func(t *testing.T) {
		f := newStore(t)
		d := ready(t, f)
		if _, err := f.Store.QueueKeep(ctx, f.UserID, d.ID); err != nil {
			t.Fatal(err)
		}
		c := claim(t, f)
		if err := f.Store.FailKeep(ctx, c, dto.DraftFailure{Reason: "flaky"}); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "keeping" || got.LastError != "flaky" {
			t.Errorf("GetDraft() = %+v, want keeping with the error while backing off", got)
		}
		if _, err := f.Store.ClaimDraft(ctx); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("ClaimDraft() err = %v, want ErrNotFound while backing off", err)
		}
	})

	t.Run("a terminal keep failure returns the Draft to ready unkept with the error", func(t *testing.T) {
		f := newStore(t)
		d := ready(t, f)
		if _, err := f.Store.QueueKeep(ctx, f.UserID, d.ID); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.FailKeep(ctx, claim(t, f), dto.DraftFailure{Reason: "gone", Terminal: true}); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "ready" || got.Outcome != nil || got.LastError != "gone" {
			t.Errorf("GetDraft() = %+v, want ready, unkept, last error set", got)
		}
	})
}
