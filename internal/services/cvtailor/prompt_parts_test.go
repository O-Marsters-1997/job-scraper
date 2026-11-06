package cvtailor_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/promptparts"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

func TestPromptPartOverrides(t *testing.T) {
	t.Parallel()
	const base = "Engineer who ships."
	const putVoice = "PUT /prompt-parts/profile-redraft/voice.md"
	const delVoice = "DELETE /prompt-parts/profile-redraft/voice.md"
	accepted := profiledraft.Attempt{Text: "Engineer who ships reliable Go.", PromptKey: "k", Model: "m"}
	setup := func(t *testing.T) (draftEnv, string, *cvtailortest.RecordingProfileWriter, http.Handler) {
		t.Helper()
		e := newDraftEnv(t)
		id, _ := e.readyWithProfile(t, base)
		w := cvtailortest.Redrafting(profiledraft.Result{Text: accepted.Text, Attempts: []profiledraft.Attempt{accepted}}, nil)
		r := newRouter(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive, Profile: w, Creds: apiKey})
		return e, id, w, r
	}
	lab := func(t *testing.T, r http.Handler, id string) map[string]dto.PromptPart {
		t.Helper()
		got := handlerstest.Do[dto.PromptLab](t, r, http.StatusOK, "GET /tailoring/drafts/"+id+"/prompt", "")
		parts := map[string]dto.PromptPart{}
		for _, p := range got.Parts {
			parts[p.Path] = p
		}
		return parts
	}

	t.Run("a saved file is used by the next Redraft and read back as yours", func(t *testing.T) {
		t.Parallel()
		_, id, w, r := setup(t)

		saved := handlerstest.Do[dto.PromptPart](t, r, http.StatusOK, putVoice, `{"text":"Write like a pirate."}`)
		handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+id+"/profile/redraft", `{}`)

		if saved.Source != "yours" {
			t.Errorf("saved source = %q, want yours", saved.Source)
		}
		if got := w.Inputs[0].Parts["voice.md"]; got != "Write like a pirate." {
			t.Errorf("Redraft voice.md = %q, want the saved text", got)
		}
		if p := lab(t, r, id)["voice.md"]; p.Source != "yours" || p.Text != "Write like a pirate." {
			t.Errorf("lab voice.md = %+v, want yours with the saved text", p)
		}
	})

	t.Run("another user's saved file does not reach this user's Redraft", func(t *testing.T) {
		t.Parallel()
		e, id, w, r := setup(t)
		if err := e.store.SavePartOverride(t.Context(), otherUserID, promptparts.ProfileRedraft, promptparts.Override{Path: "voice.md", Text: "theirs"}); err != nil {
			t.Fatal(err)
		}

		handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+id+"/profile/redraft", `{}`)

		if got := w.Inputs[0].Parts["voice.md"]; got == "theirs" || got == "" {
			t.Errorf("Redraft voice.md = %q, want the default", got)
		}
		if p := lab(t, r, id)["voice.md"]; p.Source != "default" {
			t.Errorf("lab voice.md source = %q, want default", p.Source)
		}
	})

	t.Run("using the default deletes the saved file", func(t *testing.T) {
		t.Parallel()
		_, id, _, r := setup(t)
		handlerstest.Do[dto.PromptPart](t, r, http.StatusOK, putVoice, `{"text":"mine"}`)

		handlerstest.Do[struct{}](t, r, http.StatusNoContent, delVoice, "")

		if p := lab(t, r, id)["voice.md"]; p.Source != "default" || p.Text == "mine" {
			t.Errorf("lab voice.md = %+v, want the default again", p)
		}
	})

	t.Run("unknown paths and bad text are rejected", func(t *testing.T) {
		t.Parallel()
		_, _, _, r := setup(t)
		body := func(n int) string { return `{"text":"` + strings.Repeat("a", n) + `"}` }

		handlerstest.Do[struct{}](t, r, http.StatusNotFound, "PUT /prompt-parts/profile-redraft/nope.md", body(5))
		handlerstest.Do[struct{}](t, r, http.StatusNotFound, "PUT /prompt-parts/nope/voice.md", body(5))
		handlerstest.Do[struct{}](t, r, http.StatusNotFound, "DELETE /prompt-parts/profile-redraft/nope.md", "")
		handlerstest.Do[struct{}](t, r, http.StatusBadRequest, putVoice, body(0))
		handlerstest.Do[struct{}](t, r, http.StatusBadRequest, putVoice, body(20001))
		handlerstest.Do[dto.PromptPart](t, r, http.StatusOK, putVoice, body(20000))
	})
}
