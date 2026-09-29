package cvtailortest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// RunDraftContract proves the Draft queue behaves the same in the fake and
// the real store.
func RunDraftContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := context.Background()

	input := func(f Fixture) dto.DraftInput {
		return dto.DraftInput{JobID: f.JobID, DocID: "doc", TabID: "t.0", AchievementIDs: []string{missingID}}
	}
	create := func(t *testing.T, f Fixture) dto.Draft {
		t.Helper()
		d, err := f.Store.CreateDraft(ctx, f.UserID, input(f))
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

	t.Run("completing records the result and the Doc", func(t *testing.T) {
		f := newStore(t)
		d := create(t, f)
		c := claim(t, f)
		finding := dto.DraftFinding{Check: "skills", Severity: "info", Message: "the job asks for Rust"}
		res := dto.DraftResult{EditSet: json.RawMessage(`{"positions":[]}`), Model: "m", PromptVersion: "p", DraftDocID: "doc-copy", Cost: 0.5, Findings: []dto.DraftFinding{finding}}
		if err := f.Store.CompleteDraft(ctx, c, res); err != nil {
			t.Fatal(err)
		}
		got, _ := f.Store.GetDraft(ctx, f.UserID, d.ID)
		if got.Status != "ready" || got.DraftDocID != "doc-copy" {
			t.Errorf("GetDraft() = %+v, want ready with the Doc", got)
		}
		if diff := cmp.Diff([]dto.DraftFinding{finding}, got.Findings); diff != "" {
			t.Errorf("GetDraft().Findings mismatch (-want +got):\n%s", diff)
		}
		if err := f.Store.CompleteDraft(ctx, c, res); err == nil {
			t.Error("second CompleteDraft() err = nil, want a stale-claim error")
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
}
