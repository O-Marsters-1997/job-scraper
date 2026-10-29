package cvtailor_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

type failingInsights struct{ cvtailor.Editor }

func (failingInsights) Insights(context.Context, string, cvedit.InsightsInput) (cvedit.InsightsResult, error) {
	return cvedit.InsightsResult{}, errors.New("openrouter said no")
}

func TestModule_ExplainDraft(t *testing.T) {
	t.Parallel()

	t.Run("a model failure is logged with its cause and surfaces as upstream", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		var buf bytes.Buffer
		log := slog.New(slog.NewTextHandler(&buf, nil))
		m := cvtailor.Build(cvtailor.Deps{Store: e.store, Asker: e.asker, Editor: failingInsights{}, Creds: cvtailortest.Key(apiKey), Log: log})

		_, err := m.ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id})

		if !apperr.IsKind(err, apperr.KindUpstream) || !strings.Contains(buf.String(), "openrouter said no") {
			t.Errorf("ExplainDraft() = %v, log %q, want upstream error with the cause logged", err, buf.String())
		}
	})

	setup := func(t *testing.T) (draftEnv, string, string) {
		t.Helper()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		e.asker.answers = map[string]dto.Answer{questionPrefix + e.pos.Achievements[0].Text: {PYes: 0.6, PNo: 0.1}}
		e.store.SetAnnotations(jobID, dto.JobAnnotations{Note: "Wants   a\nmentor", Comments: []string{"Team is remote first"}})
		lean, err := e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil || len(lean) != 1 {
			t.Fatalf("DraftLean() = %+v, %v, want one bullet", lean, err)
		}
		return e, id, lean[0].SlotID
	}
	module := func(e draftEnv, editor cvtailor.Editor, creds cvtailor.Credentials) *cvtailor.Module {
		return cvtailor.Build(cvtailor.Deps{Store: e.store, Asker: e.asker, Editor: editor, Creds: creds})
	}

	t.Run("a quote found in its source is anchored, one that is not is flagged", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name         string
			source       string
			quote        string
			wantAnchored bool
		}{
			{"jd verbatim", "jd", "Go engineer", true},
			{"case differs", "jd", "go engineer", false},
			{"whitespace collapsed in the note", "note", "Wants a mentor", true},
			{"any comment counts", "comment", "remote first", true},
			{"quote from another source", "note", "Go engineer", false},
			{"empty quote", "jd", "", false},
			{"unknown source", "email", "Go engineer", false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e, id, slot := setup(t)
				editor := cvtailortest.Insighting(cvedit.InsightLine{SlotID: slot, Source: tt.source, Quote: tt.quote, Why: "fits"})

				got, err := module(e, editor, cvtailortest.Key(apiKey)).ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id})

				if err != nil || len(got.Lines) != 1 || got.Lines[0].Anchored != tt.wantAnchored {
					t.Fatalf("ExplainDraft() = %+v, %v, want anchored = %v", got, err, tt.wantAnchored)
				}
				if got.Lines[0].Lean == nil {
					t.Errorf("line lean = %v, want the bullet's lean", got.Lines[0].Lean)
				}
			})
		}
	})

	t.Run("a bullet the model skipped still gets an unanchored line", func(t *testing.T) {
		t.Parallel()
		e, id, slot := setup(t)

		got, err := module(e, cvtailortest.Insighting(), cvtailortest.Key(apiKey)).ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id})

		want := []dto.InsightLine{{SlotID: slot, Lean: got.Lines[0].Lean, Source: dto.InsightSourceJD}}
		if err != nil || cmp.Diff(want, got.Lines) != "" {
			t.Errorf("ExplainDraft() = %+v, %v, want %+v", got.Lines, err, want)
		}
	})

	t.Run("the model is given the bullets, description, note and comments", func(t *testing.T) {
		t.Parallel()
		editor := cvtailortest.Insighting()
		e, id, slot := setup(t)

		if _, err := module(e, editor, cvtailortest.Key(apiKey)).ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id}); err != nil {
			t.Fatal(err)
		}

		draft, err := e.svc.GetDraft(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatal(err)
		}
		lean := 0.5
		want := []cvedit.InsightsInput{{
			Bullets:        []cvedit.InsightBullet{{SlotID: slot, Text: draft.Content.Positions[0].Bullets[0].Text, Lean: &lean}},
			JobDescription: "We need a Go engineer.", JobNote: "Wants   a\nmentor", JobComments: []string{"Team is remote first"},
		}}
		if diff := cmp.Diff(want, editor.InsightInputs); diff != "" {
			t.Errorf("model input (-want +got):\n%s", diff)
		}
	})

	t.Run("a stored result is served without the model and goes stale on an edit", func(t *testing.T) {
		t.Parallel()
		editor := cvtailortest.Insighting()
		e, id, slot := setup(t)
		m := module(e, editor, cvtailortest.Key(apiKey))
		if _, err := m.ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id}); err != nil {
			t.Fatal(err)
		}

		fresh, err := e.svc.DraftInsights(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil || len(fresh.Lines) != 1 || fresh.Stale || len(editor.InsightInputs) != 1 {
			t.Fatalf("DraftInsights() = %+v, %v with %d model calls, want the stored line, fresh, no new call", fresh, err, len(editor.InsightInputs))
		}

		_, err = e.svc.SaveDraftSlots(t.Context(), userID, dto.DraftSlotsInput{ID: id, Slots: []dto.SlotEdit{{SlotID: slot, Text: "Rewrote the query layer"}}})
		if err != nil {
			t.Fatal(err)
		}
		stale, err := e.svc.DraftInsights(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil || !stale.Stale || len(stale.Lines) != 1 {
			t.Errorf("DraftInsights() after an edit = %+v, %v, want the old line, stale", stale, err)
		}

		e.store.SetAnnotations(jobID, dto.JobAnnotations{Note: "changed"})
		again, err := m.ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id})
		if err != nil || again.Stale {
			t.Fatalf("ExplainDraft() refresh = %+v, %v, want fresh", again, err)
		}
		if got, _ := e.svc.DraftInsights(t.Context(), userID, dto.DraftQuery{ID: id}); got.Stale {
			t.Errorf("DraftInsights() after a refresh is stale, want fresh")
		}
	})

	t.Run("the draft summarises its stored insights, and has none before an Explain", func(t *testing.T) {
		t.Parallel()
		e, id, slot := setup(t)
		before, err := e.svc.GetDraft(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil || before.Insights != nil {
			t.Fatalf("GetDraft() before Explain = %+v, %v, want no insights", before.Insights, err)
		}
		editor := cvtailortest.Insighting(cvedit.InsightLine{SlotID: slot, Source: "jd", Quote: "Go engineer", Why: "fits"})
		if _, err := module(e, editor, cvtailortest.Key(apiKey)).ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id}); err != nil {
			t.Fatal(err)
		}

		got, err := e.svc.GetDraft(t.Context(), userID, dto.DraftQuery{ID: id})
		listed, listErr := e.svc.ListJobDrafts(t.Context(), userID, dto.DraftJobQuery{JobID: jobID})

		want := &dto.InsightsSummary{Anchored: 1, Total: 1}
		if err != nil || listErr != nil || len(listed) == 0 || cmp.Diff(want, got.Insights) != "" || cmp.Diff(want, listed[0].Insights) != "" {
			t.Errorf("summaries = %+v, %+v (%v, %v), want %+v", got.Insights, listed, err, listErr, want)
		}
	})

	t.Run("a draft never explained has no lines", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]

		got, err := e.svc.DraftInsights(t.Context(), userID, dto.DraftQuery{ID: id})

		if err != nil || len(got.Lines) != 0 || got.Stale {
			t.Errorf("DraftInsights() = %+v, %v, want no lines", got, err)
		}
	})

	t.Run("a missing key asks for one and does not call the model", func(t *testing.T) {
		t.Parallel()
		editor := cvtailortest.Insighting()
		e, id, _ := setup(t)

		_, err := module(e, editor, cvtailortest.NoKey{}).ExplainDraft(t.Context(), userID, dto.ExplainDraftInput{ID: id})

		if !apperr.IsKind(err, apperr.KindUnprocessable) || len(editor.InsightInputs) != 0 {
			t.Errorf("ExplainDraft() = %v with %d model calls, want unprocessable and no call", err, len(editor.InsightInputs))
		}
	})
}
