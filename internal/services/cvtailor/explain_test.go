package cvtailor_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func TestExplain(t *testing.T) {
	e := newDraftEnv(t)
	bullet := e.pos.Achievements[0]
	in := dto.ExplainInput{JobID: jobID, AchievementID: bullet.ID}
	question := questionPrefix + bullet.Text
	module := func(editor cvtailor.Editor, creds cvtailor.Credentials, answer dto.Answer) *cvtailor.Module {
		asker := &fakeAsker{answers: map[string]dto.Answer{question: answer}}
		return cvtailor.Build(cvtailor.Deps{Store: e.store, Asker: asker, Editor: editor, Creds: creds})
	}
	low := dto.Answer{PYes: 0.1, PNo: 0.7, PNotStated: 0.2}

	t.Run("a low bullet gets Haiku's guess given the bullet, job and probabilities", func(t *testing.T) {
		editor := cvtailortest.Explaining("The job wants Kubernetes.")

		got, err := module(editor, apiKey, low).Explain(t.Context(), userID, in)

		if err != nil || got.Text != "The job wants Kubernetes." {
			t.Fatalf("Explain() = %+v, %v, want the reply", got, err)
		}
		want := []cvedit.ExplainInput{{Bullet: bullet.Text, JobDescription: "We need a Go engineer.", PYes: 0.1, PNo: 0.7, PNotStated: 0.2}}
		if diff := cmp.Diff(want, editor.ExplainInputs); diff != "" {
			t.Errorf("Explain() model input (-want +got):\n%s", diff)
		}
	})

	t.Run("an unclear bullet is explained too", func(t *testing.T) {
		editor := cvtailortest.Explaining("x")

		_, err := module(editor, apiKey, dto.Answer{PYes: 0.31, PNo: 0.34, PNotStated: 0.35}).Explain(t.Context(), userID, in)

		if err != nil || len(editor.ExplainInputs) != 1 {
			t.Errorf("Explain() = %v with %d model calls, want one call", err, len(editor.ExplainInputs))
		}
	})

	t.Run("a bullet that fits is refused without calling the model", func(t *testing.T) {
		editor := cvtailortest.Explaining("x")

		_, err := module(editor, apiKey, dto.Answer{PYes: 0.7, PNo: 0.1, PNotStated: 0.2}).Explain(t.Context(), userID, in)

		if !apperr.IsKind(err, apperr.KindConflict) || len(editor.ExplainInputs) != 0 {
			t.Errorf("Explain() = %v with %d model calls, want a conflict and no call", err, len(editor.ExplainInputs))
		}
	})

	t.Run("a missing key asks for one and does not call the model", func(t *testing.T) {
		editor := cvtailortest.Explaining("x")

		_, err := module(editor, cvtailortest.NoKey{}, low).Explain(t.Context(), userID, in)

		if !apperr.IsKind(err, apperr.KindUnprocessable) || len(editor.ExplainInputs) != 0 {
			t.Errorf("Explain() = %v with %d model calls, want unprocessable and no call", err, len(editor.ExplainInputs))
		}
	})

	t.Run("an unknown bullet is not found", func(t *testing.T) {
		_, err := module(cvtailortest.Explaining("x"), apiKey, low).Explain(t.Context(), userID, dto.ExplainInput{JobID: jobID, AchievementID: "nope"})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("Explain(unknown) = %v, want not found", err)
		}
	})
}

func TestExplainRoute(t *testing.T) {
	e := newDraftEnv(t)
	bullet := e.pos.Achievements[0]
	asker := &fakeAsker{answers: map[string]dto.Answer{questionPrefix + bullet.Text: {PYes: 0.1, PNo: 0.7, PNotStated: 0.2}}}
	r := newRouter(cvtailor.Deps{Store: e.store, Asker: asker, Editor: cvtailortest.Explaining("A guess."), Creds: apiKey})

	got := handlerstest.Do[dto.Explanation](t, r, http.StatusOK, "POST /tailoring/jobs/"+jobID+"/achievements/"+bullet.ID+"/explain", "")

	if got.Text != "A guess." {
		t.Errorf("POST explain = %+v, want the reply", got)
	}
}
