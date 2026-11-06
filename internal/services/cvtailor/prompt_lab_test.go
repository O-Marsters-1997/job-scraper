package cvtailor_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
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

func TestPromptTryRoute(t *testing.T) {
	t.Parallel()
	const base = "Engineer who ships."
	setup := func(t *testing.T, w *cvtailortest.RecordingProfileWriter, creds cvtailor.Credentials) (draftEnv, string, chi.Router) {
		t.Helper()
		e := newDraftEnv(t)
		id, _ := e.readyWithProfile(t, base)
		return e, id, newRouter(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive, Profile: w, Creds: creds})
	}
	route := func(id string) string { return "POST /tailoring/drafts/" + id + "/prompt/try" }

	t.Run("edited Parts stream deltas then done and save nothing", func(t *testing.T) {
		t.Parallel()
		w := cvtailortest.Trying(profiledraft.TryResult{Text: "Engineer who ships Go.", Cost: 0.02, PromptKey: "k"}, nil, "Engineer ", "who ships Go.")
		e, id, r := setup(t, w, apiKey)

		got := handlerstest.Serve(t, r, route(id), `{"parts":{"voice.md":"Be brief."},"steer":"calmer"}`)

		want := "event: delta\ndata: {\"text\":\"Engineer \"}\n\n" +
			"event: delta\ndata: {\"text\":\"who ships Go.\"}\n\n" +
			"event: done\ndata: {\"text\":\"Engineer who ships Go.\",\"findings\":[],\"cost\":0.02,\"promptKey\":\"k\"}\n\n"
		if diff := cmp.Diff(want, got.Body.String()); diff != "" {
			t.Errorf("try body (-want +got):\n%s", diff)
		}
		if len(w.TryInputs) != 1 || w.TryInputs[0].Parts["voice.md"] != "Be brief." || w.TryInputs[0].Steer != "calmer" {
			t.Errorf("try inputs = %+v, want the edited Part and steer", w.TryInputs)
		}
		if p := e.draft(t, id).Content.Profile; p == nil || *p != base {
			t.Errorf("profile = %v, want unchanged %q", p, base)
		}
		if n := len(e.store.ProfileAttempts(id)); n != 0 {
			t.Errorf("recorded %d Attempts, want 0", n)
		}
	})

	t.Run("a blocking finding rides on done", func(t *testing.T) {
		t.Parallel()
		res := profiledraft.TryResult{Text: "x", Findings: []checks.Finding{{Check: "grounding", Severity: checks.Block, Message: "unsupported"}}}
		_, id, r := setup(t, cvtailortest.Trying(res, nil), apiKey)

		got := handlerstest.Serve(t, r, route(id), `{}`)

		if !strings.Contains(got.Body.String(), `"findings":[{"check":"grounding"`) {
			t.Errorf("try body = %q, want the finding on done", got.Body)
		}
	})

	for name, tc := range map[string]struct {
		body  string
		creds cvtailor.Credentials
		want  int
	}{
		"no OpenRouter key": {`{}`, cvtailortest.NoKey{}, http.StatusUnprocessableEntity},
		"unknown Part path": {`{"parts":{"nope.md":"x"}}`, apiKey, http.StatusBadRequest},
		"oversized Part":    {`{"parts":{"voice.md":"` + strings.Repeat("x", 20001) + `"}}`, apiKey, http.StatusBadRequest},
		"multi-line steer":  {`{"steer":"a\nb"}`, apiKey, http.StatusBadRequest},
	} {
		t.Run("refused before streaming: "+name, func(t *testing.T) {
			t.Parallel()
			w := cvtailortest.Trying(profiledraft.TryResult{}, nil)
			_, id, r := setup(t, w, tc.creds)

			got := handlerstest.Serve(t, r, route(id), tc.body)

			if ct := got.Header().Get("Content-Type"); got.Code != tc.want || !strings.HasPrefix(ct, "application/json") {
				t.Errorf("try = %d %q, want %d JSON error", got.Code, ct, tc.want)
			}
			if len(w.TryInputs) != 0 {
				t.Errorf("model was called %d times, want 0", len(w.TryInputs))
			}
		})
	}
}

func TestUseProfileTryRoute(t *testing.T) {
	t.Parallel()
	const base = "Engineer who ships."
	setup := func(t *testing.T) (draftEnv, string, chi.Router) {
		t.Helper()
		e := newDraftEnv(t)
		id, _ := e.readyWithProfile(t, base)
		return e, id, newRouter(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive, Profile: cvtailortest.Trying(profiledraft.TryResult{}, nil), Creds: apiKey})
	}
	route := func(id string) string { return "POST /tailoring/drafts/" + id + "/profile/use" }

	t.Run("a clean Try replaces the Profile and records one Attempt keyed like the Try", func(t *testing.T) {
		t.Parallel()
		e, id, r := setup(t)
		body := `{"text":"Engineer who ships well.","steer":"calmer","parts":{"voice.md":"Be brief."}}`

		got := handlerstest.Do[dto.Draft](t, r, http.StatusOK, route(id), body)

		if got.Content == nil || got.Content.Profile == nil || *got.Content.Profile != "Engineer who ships well." {
			t.Errorf("content.profile = %+v, want the Try's text", got.Content)
		}
		attempts := e.store.ProfileAttempts(id)
		if len(attempts) != 1 || attempts[0].Steer != "calmer" || attempts[0].Rejection != "" || attempts[0].PromptKey == "" {
			t.Fatalf("Attempts = %+v, want exactly one clean Attempt with the steer and a prompt key", attempts)
		}
		handlerstest.Do[dto.Draft](t, r, http.StatusOK, route(id), `{"text":"Engineer who ships well.","steer":"calmer"}`)
		again := e.store.ProfileAttempts(id)
		if len(again) != 2 {
			t.Fatalf("recorded %d Attempts, want 2", len(again))
		}
		if again[0].PromptKey == again[1].PromptKey {
			t.Errorf("prompt key %q is the same for edited and default Parts", again[0].PromptKey)
		}
	})

	t.Run("a blocking finding is a 422 and the Draft is unchanged", func(t *testing.T) {
		t.Parallel()
		e, id, r := setup(t)

		handlerstest.Do[struct{}](t, r, http.StatusUnprocessableEntity, route(id), `{"text":"Engineer who ships Go."}`)

		if p := e.draft(t, id).Content.Profile; p == nil || *p != base {
			t.Errorf("profile = %v, want unchanged %q", p, base)
		}
		if n := len(e.store.ProfileAttempts(id)); n != 0 {
			t.Errorf("recorded %d Attempts, want 0", n)
		}
	})

	t.Run("a discarded Draft is a 409", func(t *testing.T) {
		t.Parallel()
		_, id, r := setup(t)
		handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+id+"/discard", "")

		handlerstest.Do[struct{}](t, r, http.StatusConflict, route(id), `{"text":"Engineer who ships well."}`)
	})
}
