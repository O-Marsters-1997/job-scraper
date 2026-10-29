package cvtailor_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

func TestRedraftProfileRoute(t *testing.T) {
	t.Parallel()
	const base = "Engineer who ships."
	accepted := profiledraft.Attempt{Text: "Engineer who ships reliable Go services.", Steer: "calmer", PromptKey: "k", Model: "m", Cost: 0.01}
	rejected := func(text string) profiledraft.Attempt {
		return profiledraft.Attempt{Text: text, PromptKey: "k", Model: "m", Rejection: "uses a banned word"}
	}

	t.Run("a Draft whose CV has a Skills section still redrafts", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		docs := cvtailortest.Docs{TabJSON: tabJSON(t, head("Profile"), prose(base), head(heading), bullet("Built and maintained the public APIs for the platform"), head("Skills"), prose("Go, SQL"))}
		edits := e.citedEdit("Cut p99 latency by moving queries to Postgres")
		profile := base
		edits.Profile = &profile
		id := e.readyWith(t, tick{docs: docs}, edits)
		w := cvtailortest.Redrafting(profiledraft.Result{Text: accepted.Text, Attempts: []profiledraft.Attempt{accepted}}, nil)
		r := newRouter(cvtailor.Deps{Store: e.store, Docs: docs, Drive: e.drive, Profile: w, Creds: apiKey})

		got := handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+id+"/profile/redraft", `{}`)

		if got.Content == nil || got.Content.Profile == nil || *got.Content.Profile != accepted.Text {
			t.Errorf("redraft content.profile = %+v, want %q", got.Content, accepted.Text)
		}
	})

	type env struct {
		e      draftEnv
		id     string
		router chi.Router
		writer *cvtailortest.RecordingProfileWriter
	}
	setup := func(t *testing.T, res profiledraft.Result, werr error, creds cvtailor.Credentials) env {
		t.Helper()
		e := newDraftEnv(t)
		id, _ := e.readyWithProfile(t, base)
		w := cvtailortest.Redrafting(res, werr)
		r := newRouter(cvtailor.Deps{Store: e.store, Docs: profileTab(t), Drive: e.drive, Profile: w, Creds: creds})
		return env{e: e, id: id, router: r, writer: w}
	}
	route := func(id string) string { return "POST /tailoring/drafts/" + id + "/profile/redraft" }

	t.Run("an accepted redraft is saved and every Attempt recorded", func(t *testing.T) {
		t.Parallel()
		s := setup(t, profiledraft.Result{Text: accepted.Text, Attempts: []profiledraft.Attempt{rejected("Leverage it"), accepted}}, nil, apiKey)

		got := handlerstest.Do[dto.Draft](t, s.router, http.StatusOK, route(s.id), `{"steer":"calmer"}`)

		if got.Content == nil || got.Content.Profile == nil || *got.Content.Profile != accepted.Text {
			t.Errorf("redraft content.profile = %+v, want %q", got.Content, accepted.Text)
		}
		want := []dto.ProfileAttempt{
			{Text: "Leverage it", PromptKey: "k", Model: "m", Rejection: "uses a banned word"},
			{Text: accepted.Text, Steer: "calmer", PromptKey: "k", Model: "m", Cost: 0.01},
		}
		if diff := cmp.Diff(want, s.e.store.ProfileAttempts(s.id)); diff != "" {
			t.Errorf("recorded Attempts (-want +got):\n%s", diff)
		}
		if len(s.writer.Inputs) != 1 || s.writer.Inputs[0].Profile != base || s.writer.Inputs[0].Steer != "calmer" ||
			s.writer.Inputs[0].JobTitle != "Platform Engineer" || len(s.writer.Inputs[0].Bank) != 2 {
			t.Errorf("writer inputs = %+v, want the base profile, steer, job and bank", s.writer.Inputs)
		}
	})

	t.Run("two rejected Attempts leave the Profile unchanged and are recorded", func(t *testing.T) {
		t.Parallel()
		s := setup(t, profiledraft.Result{Attempts: []profiledraft.Attempt{rejected("one"), rejected("two")}}, nil, apiKey)

		w := handlerstest.Serve(t, s.router, route(s.id), `{}`)

		if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "the redraft broke the rules: uses a banned word") {
			t.Errorf("redraft = %d %s, want 422 naming the rejection", w.Code, w.Body)
		}
		if n := len(s.e.store.ProfileAttempts(s.id)); n != 2 {
			t.Errorf("recorded %d Attempts, want 2", n)
		}
		if p := s.e.draft(t, s.id).Content.Profile; p == nil || *p != base {
			t.Errorf("profile = %v, want unchanged %q", p, base)
		}
	})

	t.Run("a writer error is a 502", func(t *testing.T) {
		t.Parallel()
		s := setup(t, profiledraft.Result{}, errors.New("boom"), apiKey)

		if w := handlerstest.Serve(t, s.router, route(s.id), `{}`); w.Code != http.StatusBadGateway {
			t.Errorf("redraft = %d, want 502", w.Code)
		}
	})

	t.Run("no OpenRouter key is a 422", func(t *testing.T) {
		t.Parallel()
		s := setup(t, profiledraft.Result{}, nil, cvtailortest.NoKey{})

		if w := handlerstest.Serve(t, s.router, route(s.id), `{}`); w.Code != http.StatusUnprocessableEntity {
			t.Errorf("redraft = %d, want 422", w.Code)
		}
	})

	for name, steer := range map[string]string{
		"multi-line": `{"steer":"a\nb"}`,
		"over-long":  `{"steer":"` + strings.Repeat("a", 201) + `"}`,
	} {
		t.Run("a "+name+" steer is a 400", func(t *testing.T) {
			t.Parallel()
			s := setup(t, profiledraft.Result{}, nil, apiKey)

			if w := handlerstest.Serve(t, s.router, route(s.id), steer); w.Code != http.StatusBadRequest {
				t.Errorf("redraft = %d, want 400", w.Code)
			}
		})
	}

	t.Run("a discarded Draft is a 409", func(t *testing.T) {
		t.Parallel()
		s := setup(t, profiledraft.Result{}, nil, apiKey)
		handlerstest.Do[dto.Draft](t, s.router, http.StatusOK, "POST /tailoring/drafts/"+s.id+"/discard", "")

		if w := handlerstest.Serve(t, s.router, route(s.id), `{}`); w.Code != http.StatusConflict {
			t.Errorf("redraft = %d, want 409", w.Code)
		}
	})

	t.Run("a Draft that is not ready is a 409", func(t *testing.T) {
		t.Parallel()
		s := setup(t, profiledraft.Result{}, nil, apiKey)
		queued := s.e.queueOnOtherJob(t)

		if w := handlerstest.Serve(t, s.router, route(queued), `{}`); w.Code != http.StatusConflict {
			t.Errorf("redraft = %d, want 409", w.Code)
		}
	})
}
