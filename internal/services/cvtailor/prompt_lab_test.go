package cvtailor_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

func TestPromptLab(t *testing.T) {
	t.Parallel()
	const base = "Engineer who ships."
	setup := func(t *testing.T) (draftEnv, string, *cvtailor.Module) {
		t.Helper()
		e := newDraftEnv(t)
		id, _ := e.readyWithProfile(t, base)
		m := cvtailor.Build(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive})
		return e, id, m
	}

	t.Run("a ready Draft returns the Parts, documents and checks", func(t *testing.T) {
		t.Parallel()
		e, id, _ := setup(t)
		r := newRouter(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive, Profile: cvtailortest.Redrafting(profiledraft.Result{}, nil)})

		got := handlerstest.Do[dto.PromptLab](t, r, http.StatusOK, "GET /tailoring/drafts/"+id+"/prompt", "")

		if got.Main != "redraft-profile.md" || len(got.Parts) != 2 || len(got.Documents) != 2 || len(got.Checks) == 0 {
			t.Fatalf("lab = %+v, want main redraft-profile.md, two Parts, two documents and checks", got)
		}
		if p := got.Documents[1]; p.Path != "profile.md" || !strings.Contains(p.Text, base) {
			t.Errorf("profile document = %+v, want the base profile", p)
		}
		if strings.Contains(got.Parts[0].Text, "<documents>") {
			t.Errorf("lab leaks the assembled message: %q", got.Parts[0].Text)
		}
	})

	t.Run("another user's Draft is not found", func(t *testing.T) {
		t.Parallel()
		_, id, m := setup(t)

		_, err := m.PromptLab(t.Context(), otherUserID, dto.PromptLabQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("PromptLab(other user) error = %v, want not found", err)
		}
	})

	t.Run("a discarded Draft is a 409", func(t *testing.T) {
		t.Parallel()
		e, id, _ := setup(t)
		r := newRouter(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive})
		handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+id+"/discard", "")

		handlerstest.Do[struct{}](t, r, http.StatusConflict, "GET /tailoring/drafts/"+id+"/prompt", "")
	})
}
