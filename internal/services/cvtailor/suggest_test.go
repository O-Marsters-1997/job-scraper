package cvtailor_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func (e draftEnv) suggestModule(editor cvtailor.Editor, creds cvtailor.Credentials) *cvtailor.Module {
	return cvtailor.Build(cvtailor.Deps{Store: e.store, Drive: e.drive, Editor: editor, Creds: creds})
}

func collect(t *testing.T, m *cvtailor.Module, in dto.SuggestInput) ([]cvtailor.SuggestEvent, error) {
	t.Helper()
	events, err := m.Suggest(t.Context(), userID, in)
	if err != nil {
		return nil, err
	}
	return slices.Collect(events), nil
}

func TestSuggest(t *testing.T) {
	t.Run("streams deltas then a done event with the grounding findings", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		m := e.suggestModule(cvtailortest.Suggesting("Leveraged ", "Kubernetes"), apiKey)

		got, err := collect(t, m, dto.SuggestInput{ID: id, SlotID: slot, Action: "tighten", Text: "Cut p99 latency"})

		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(got) != 3 || got[0].Delta != "Leveraged " || got[1].Delta != "Kubernetes" || got[2].Done == nil {
			t.Fatalf("Suggest() events = %+v, want two deltas then done", got)
		}
		done := got[2].Done
		if done.Text != "Leveraged Kubernetes" {
			t.Errorf("done text = %q, want the whole suggestion", done.Text)
		}
		checks := map[string]bool{}
		for _, f := range done.Findings {
			if f.SlotID != slot {
				t.Errorf("finding %+v is not on slot %q", f, slot)
			}
			checks[f.Check] = true
		}
		if !checks["banned_words"] || !checks["grounding"] {
			t.Errorf("done findings = %+v, want banned_words and grounding", done.Findings)
		}
	})

	t.Run("a bullet draws on its cited Achievements and the profile on the Draft's chosen ones", func(t *testing.T) {
		e := newDraftEnv(t)
		e.input.AchievementIDs = []string{e.pos.Achievements[0].ID, e.pos.Achievements[1].ID}
		id, _ := e.readyWithProfile(t, "Engineer who ships.")
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		editor := cvtailortest.Suggesting("ok")
		m := e.suggestModule(editor, apiKey)

		for _, slotID := range []string{slot, "profile"} {
			if _, err := collect(t, m, dto.SuggestInput{ID: id, SlotID: slotID, Action: "verb", Text: "Engineer who ships."}); err != nil {
				t.Fatalf("Suggest(%s) error = %v", slotID, err)
			}
		}

		bullet, profile := editor.SuggestInputs[0], editor.SuggestInputs[1]
		if diff := cmp.Diff([]string{e.pos.Achievements[0].Text}, bullet.Achievements); diff != "" {
			t.Errorf("bullet Achievements (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{e.pos.Achievements[0].Text, e.pos.Achievements[1].Text}, profile.Achievements); diff != "" {
			t.Errorf("profile Achievements (-want +got):\n%s", diff)
		}
	})

	t.Run("refuses before streaming", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		pending := e.queue(t)
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		ok := dto.SuggestInput{ID: id, SlotID: slot, Action: "tighten", Text: "Cut p99 latency"}
		withInput := func(mutate func(*dto.SuggestInput)) dto.SuggestInput { in := ok; mutate(&in); return in }
		cases := []struct {
			name  string
			creds cvtailor.Credentials
			in    dto.SuggestInput
			kind  apperr.Kind
		}{
			{"unknown slot is not found", apiKey, dto.SuggestInput{ID: id, SlotID: "s99", Action: "tighten", Text: "x"}, apperr.KindNotFound},
			{"profile of a CV without one is not found", apiKey, dto.SuggestInput{ID: id, SlotID: "profile", Action: "tighten", Text: "x"}, apperr.KindNotFound},
			{"draft that is not ready is a conflict", apiKey, dto.SuggestInput{ID: pending, SlotID: slot, Action: "tighten", Text: "x"}, apperr.KindConflict},
			{"missing draft is not found", apiKey, dto.SuggestInput{ID: "nope", SlotID: slot, Action: "tighten", Text: "x"}, apperr.KindNotFound},
			{"no OpenRouter key is unprocessable", cvtailortest.NoKey{}, ok, apperr.KindUnprocessable},
			{"unknown action is invalid", apiKey, withInput(func(in *dto.SuggestInput) { in.Action = "rewrite" }), apperr.KindInvalid},
			{"blank text is invalid", apiKey, withInput(func(in *dto.SuggestInput) { in.Text = " " }), apperr.KindInvalid},
			{"fit without a limit is invalid", apiKey, withInput(func(in *dto.SuggestInput) { in.Action = "fit" }), apperr.KindInvalid},
			{"ask without a prompt is invalid", apiKey, withInput(func(in *dto.SuggestInput) { in.Action = "ask" }), apperr.KindInvalid},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				editor := cvtailortest.Suggesting("x")

				_, err := collect(t, e.suggestModule(editor, tc.creds), tc.in)

				if !apperr.IsKind(err, tc.kind) {
					t.Errorf("Suggest(%+v) error = %v, want kind %v", tc.in, err, tc.kind)
				}
				if len(editor.SuggestInputs) != 0 {
					t.Errorf("Suggest(%+v) called the model, want a refusal first", tc.in)
				}
			})
		}
	})

	t.Run("an upstream failure before the first word is an error, not a stream", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		m := e.suggestModule(cvtailortest.SuggestFailing(&openrouter.StatusError{Code: http.StatusPaymentRequired}), apiKey)

		_, err := collect(t, m, dto.SuggestInput{ID: id, SlotID: slot, Action: "tighten", Text: "x"})

		if !apperr.IsKind(err, apperr.KindUnprocessable) {
			t.Errorf("Suggest() error = %v, want an unprocessable error for a 402", err)
		}
	})

	t.Run("a failure after the first word ends the stream with an error event", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
		m := e.suggestModule(cvtailortest.SuggestFailing(&openrouter.StatusError{Code: 502}, "Cut"), apiKey)

		got, err := collect(t, m, dto.SuggestInput{ID: id, SlotID: slot, Action: "tighten", Text: "x"})

		if err != nil {
			t.Fatalf("Suggest() error = %v", err)
		}
		if len(got) != 2 || got[0].Delta != "Cut" || got[1].Err == nil || got[1].Done != nil {
			t.Errorf("Suggest() events = %+v, want a delta then an error", got)
		}
	})
}

func TestSuggestRoute(t *testing.T) {
	e := newDraftEnv(t)
	id := e.readyDrafts(t, 1)[0]
	slot := e.draft(t, id).Provenance.Positions[0].Bullets[0].SlotID
	body := `{"action":"tighten","text":"Cut p99 latency"}`

	t.Run("streams delta events then done as text/event-stream", func(t *testing.T) {
		r := newRouter(cvtailor.Deps{Store: e.store, Editor: cvtailortest.Suggesting("Cut ", "latency"), Creds: apiKey})

		w := handlerstest.Serve(t, r, "POST /tailoring/drafts/"+id+"/slots/"+slot+"/suggest", body)

		if ct := w.Header().Get("Content-Type"); w.Code != http.StatusOK || !strings.HasPrefix(ct, "text/event-stream") {
			t.Fatalf("POST suggest = %d %q, want 200 text/event-stream", w.Code, ct)
		}
		want := "event: delta\ndata: {\"text\":\"Cut \"}\n\n" +
			"event: delta\ndata: {\"text\":\"latency\"}\n\n" +
			"event: done\ndata: {\"text\":\"Cut latency\",\"findings\":[]}\n\n"
		if diff := cmp.Diff(want, w.Body.String()); diff != "" {
			t.Errorf("POST suggest body (-want +got):\n%s", diff)
		}
	})

	t.Run("a failure before the stream opens is a JSON error", func(t *testing.T) {
		r := newRouter(cvtailor.Deps{Store: e.store, Editor: cvtailortest.Suggesting("x"), Creds: cvtailortest.NoKey{}})

		w := handlerstest.Serve(t, r, "POST /tailoring/drafts/"+id+"/slots/"+slot+"/suggest", body)

		if ct := w.Header().Get("Content-Type"); w.Code != http.StatusUnprocessableEntity || !strings.HasPrefix(ct, "application/json") {
			t.Errorf("POST suggest without a key = %d %q, want a 422 JSON error", w.Code, ct)
		}
	})

	t.Run("the path ids win over the body's", func(t *testing.T) {
		editor := cvtailortest.Suggesting("x")
		r := newRouter(cvtailor.Deps{Store: e.store, Editor: editor, Creds: apiKey})

		w := handlerstest.Serve(t, r, "POST /tailoring/drafts/"+id+"/slots/"+slot+"/suggest", `{"id":"other","slotId":"nope","action":"tighten","text":"Cut"}`)

		if w.Code != http.StatusOK {
			t.Errorf("POST suggest = %d, want 200 using the path ids: %s", w.Code, w.Body)
		}
	})
}
