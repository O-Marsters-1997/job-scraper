package cvtailor_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/letterwrite"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type letterEnv struct {
	draftEnv
	in dto.ScaffoldInput
}

func newLetterEnv(t *testing.T) letterEnv {
	t.Helper()
	e := newDraftEnv(t)
	e.store.TrackLetterTab(userID, docID, tabID)
	if _, err := e.store.SaveLetterSkeleton(t.Context(), userID, dto.LetterSkeleton{Intro: "Dear team,", Conclusion: "Thanks, Sam", Sections: []string{"fit", "why us"}}); err != nil {
		t.Fatal(err)
	}
	return letterEnv{draftEnv: e, in: dto.ScaffoldInput{JobID: jobID, DocID: docID, TabID: tabID, AchievementIDs: []string{e.pos.Achievements[0].ID}}}
}

func (e letterEnv) module(t *testing.T, scaffolder cvtailor.Scaffolder, creds cvtailor.Credentials) *cvtailor.Module {
	t.Helper()
	docs := cvtailortest.Docs{TabJSON: tabJSON(t, prose("I led a platform team."), prose("I want to join you."))}
	return cvtailor.Build(cvtailor.Deps{Store: e.store, Docs: docs, Drive: e.drive, Creds: creds, Scaffolder: scaffolder})
}

func scaffoldOf(achievementID string) letterwrite.Result {
	return letterwrite.Result{Cost: 0.01, Scaffold: letterwrite.Scaffold{Paragraphs: []letterwrite.Paragraph{
		{SourceIndex: 0, Purpose: "fit", Bullets: []letterwrite.Bullet{
			{Text: "Cut latency", AchievementIDs: []string{achievementID}},
			{Text: "No Kubernetes yet", Gap: true},
		}},
		{SourceIndex: 1, Purpose: "why us"},
	}}}
}

func TestScaffold(t *testing.T) {
	t.Parallel()
	t.Run("writes one tab laid out as intro, paragraphs with bullets, conclusion and records it", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		got, err := e.module(t, cvtailortest.Scaffolding(scaffoldOf(e.in.AchievementIDs[0])), apiKey).Scaffold(t.Context(), userID, e.in)

		if err != nil {
			t.Fatalf("Scaffold() error = %v", err)
		}
		if len(e.drive.AddedTabs) != 1 || len(e.drive.Updates) != 1 || len(e.drive.Updates[0]) != 1 {
			t.Fatalf("drive saw %d tabs and %d updates, want one of each", len(e.drive.AddedTabs), len(e.drive.Updates))
		}
		var req struct {
			InsertText struct {
				Location struct {
					Index int    `json:"index"`
					TabID string `json:"tabId"`
				} `json:"location"`
				Text string `json:"text"`
			} `json:"insertText"`
		}
		if err := json.Unmarshal(e.drive.Updates[0][0], &req); err != nil {
			t.Fatal(err)
		}
		want := "Dear team,\nI led a platform team.\n• Cut latency\n• [GAP] No Kubernetes yet\nI want to join you.\nThanks, Sam"
		if req.InsertText.Text != want || req.InsertText.Location.TabID != "t.added1" || req.InsertText.Location.Index != 1 {
			t.Errorf("insertText = %+v, want text %q into tab t.added1 at 1", req.InsertText, want)
		}
		if got.SourceDocID != docID || got.SourceTabID != tabID || !strings.Contains(got.TabURL, "tab=t.added1") || got.Cost != 0.01 {
			t.Errorf("Scaffold() = %+v, want the source, scaffold tab url and cost", got)
		}
		listed, err := e.store.ListJobCoverLetters(t.Context(), userID, jobID)
		if err != nil || len(listed) != 1 || listed[0].ID != got.ID {
			t.Errorf("ListJobCoverLetters() = %+v, %v, want the new letter", listed, err)
		}
	})

	t.Run("the model sees the job, skeleton sections, source paragraphs and chosen achievements", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		sc := cvtailortest.Scaffolding(scaffoldOf(e.in.AchievementIDs[0]))

		if _, err := e.module(t, sc, apiKey).Scaffold(t.Context(), userID, e.in); err != nil {
			t.Fatal(err)
		}

		want := []letterwrite.Input{{
			JobDescription: "We need a Go engineer.", Company: "Acme", Role: "Platform Engineer",
			Sections:         []string{"fit", "why us"},
			SourceParagraphs: []string{"I led a platform team.", "I want to join you."},
			Achievements:     []letterwrite.Achievement{{ID: e.pos.Achievements[0].ID, Text: e.pos.Achievements[0].Text, Context: e.pos.Achievements[0].Context}},
		}}
		if diff := cmp.Diff(want, sc.Inputs); diff != "" {
			t.Errorf("model input (-want +got):\n%s", diff)
		}
	})

	t.Run("a citation outside the input is retried once and the costs add up", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		sc := cvtailortest.Scaffolding(scaffoldOf("not-chosen"), scaffoldOf(e.in.AchievementIDs[0]))

		got, err := e.module(t, sc, apiKey).Scaffold(t.Context(), userID, e.in)

		if err != nil || len(sc.Inputs) != 2 || got.Cost != 0.02 {
			t.Errorf("Scaffold() = %+v, %v after %d calls, want success on the second call costing 0.02", got, err, len(sc.Inputs))
		}
	})

	t.Run("a second bad citation fails without writing", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		sc := cvtailortest.Scaffolding(scaffoldOf("not-chosen"))

		_, err := e.module(t, sc, apiKey).Scaffold(t.Context(), userID, e.in)

		if !apperr.IsKind(err, apperr.KindUpstream) || len(sc.Inputs) != 2 || len(e.drive.AddedTabs) != 0 {
			t.Errorf("Scaffold() = %v after %d calls, %d tabs, want upstream after two calls and no tab", err, len(sc.Inputs), len(e.drive.AddedTabs))
		}
	})

	t.Run("a source index outside the letter fails without a retry", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		bad := scaffoldOf(e.in.AchievementIDs[0])
		bad.Scaffold.Paragraphs[0].SourceIndex = 9
		sc := cvtailortest.Scaffolding(bad)

		_, err := e.module(t, sc, apiKey).Scaffold(t.Context(), userID, e.in)

		if !apperr.IsKind(err, apperr.KindUpstream) || len(sc.Inputs) != 1 {
			t.Errorf("Scaffold() = %v after %d calls, want upstream after one", err, len(sc.Inputs))
		}
	})

	t.Run("rejected inputs never reach the model", func(t *testing.T) {
		t.Parallel()
		untracked := func(in *dto.ScaffoldInput) { in.TabID = "t.other" }
		tests := []struct {
			name string
			edit func(*dto.ScaffoldInput)
			kind apperr.Kind
		}{
			{"a tab that is not a tracked letter", untracked, apperr.KindInvalid},
			{"no achievements", func(in *dto.ScaffoldInput) { in.AchievementIDs = nil }, apperr.KindInvalid},
			{"a repeated achievement", func(in *dto.ScaffoldInput) { in.AchievementIDs = append(in.AchievementIDs, in.AchievementIDs[0]) }, apperr.KindInvalid},
			{"an unknown achievement", func(in *dto.ScaffoldInput) { in.AchievementIDs = []string{"nope"} }, apperr.KindInvalid},
			{"a missing tab id", func(in *dto.ScaffoldInput) { in.TabID = "" }, apperr.KindInvalid},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e := newLetterEnv(t)
				sc := cvtailortest.Scaffolding(scaffoldOf("x"))
				in := e.in
				tt.edit(&in)

				_, err := e.module(t, sc, apiKey).Scaffold(t.Context(), userID, in)

				if !apperr.IsKind(err, tt.kind) || len(sc.Inputs) != 0 {
					t.Errorf("Scaffold() = %v with %d model calls, want %v and none", err, len(sc.Inputs), tt.kind)
				}
			})
		}
	})

	t.Run("a missing key asks for one", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		sc := cvtailortest.Scaffolding(scaffoldOf("x"))

		_, err := e.module(t, sc, cvtailortest.NoKey{}).Scaffold(t.Context(), userID, e.in)

		if !apperr.IsKind(err, apperr.KindUnprocessable) || len(sc.Inputs) != 0 {
			t.Errorf("Scaffold() = %v with %d model calls, want unprocessable and none", err, len(sc.Inputs))
		}
	})

	t.Run("missing Google write access surfaces the reconnect error and records nothing", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		e.drive.AddTabErr = google.ErrNoWriteAccess

		_, err := e.module(t, cvtailortest.Scaffolding(scaffoldOf(e.in.AchievementIDs[0])), apiKey).Scaffold(t.Context(), userID, e.in)

		listed, _ := e.store.ListJobCoverLetters(t.Context(), userID, jobID)
		if !errors.Is(err, google.ErrNoWriteAccess) || len(listed) != 0 {
			t.Errorf("Scaffold() = %v with %d letters, want ErrNoWriteAccess and none", err, len(listed))
		}
	})

	t.Run("a failed write removes the half-made tab", func(t *testing.T) {
		t.Parallel()
		e := newLetterEnv(t)
		drive := cvtailortest.FailsFirstBatchUpdate(e.drive, errors.New("docs down"))
		m := cvtailor.Build(cvtailor.Deps{Store: e.store, Docs: cvtailortest.Docs{TabJSON: tabJSON(t, prose("Hello."))}, Drive: drive, Creds: apiKey,
			Scaffolder: cvtailortest.Scaffolding(letterwrite.Result{Scaffold: letterwrite.Scaffold{Paragraphs: []letterwrite.Paragraph{{SourceIndex: 0}}}})})

		_, err := m.Scaffold(t.Context(), userID, e.in)

		if err == nil || len(e.drive.Updates) != 1 || !strings.Contains(string(e.drive.Updates[0][0]), `"deleteTab"`) {
			t.Errorf("Scaffold() = %v with updates %v, want an error and a deleteTab", err, e.drive.Updates)
		}
	})
}

func TestScaffoldRoutes(t *testing.T) {
	t.Parallel()
	e := newLetterEnv(t)
	docs := cvtailortest.Docs{TabJSON: tabJSON(t, prose("I led a platform team."), prose("I want to join you."))}
	r := newRouter(cvtailor.Deps{Store: e.store, Docs: docs, Drive: e.drive, Creds: apiKey, Scaffolder: cvtailortest.Scaffolding(scaffoldOf(e.in.AchievementIDs[0]))})
	body := `{"docId":"` + docID + `","tabId":"` + tabID + `","achievementIds":["` + e.in.AchievementIDs[0] + `"]}`

	created := handlerstest.Do[dto.CoverLetter](t, r, http.StatusCreated, "POST /tailoring/jobs/"+jobID+"/cover-letters", body)
	listed := handlerstest.Do[[]dto.CoverLetter](t, r, http.StatusOK, "GET /tailoring/jobs/"+jobID+"/cover-letters", "")

	if len(listed) != 1 || listed[0].ID != created.ID || created.JobID != jobID {
		t.Errorf("POST then GET cover-letters = %+v, %+v, want the one created letter", created, listed)
	}
	handlerstest.RequiresAuth(t, r, "POST /tailoring/jobs/"+jobID+"/cover-letters", "GET /tailoring/jobs/"+jobID+"/cover-letters")

	checked := handlerstest.Do[dto.LetterCheck](t, r, http.StatusOK, "GET /tailoring/cover-letters/"+created.ID+"/check", "")
	if checked.Words != 10 || len(checked.Findings) == 0 || checked.Findings[0].Kind != dto.FindingLength {
		t.Errorf("GET check = %+v, want the 10 words of the tab and a length finding", checked)
	}
	done := handlerstest.Do[dto.CoverLetter](t, r, http.StatusOK, "POST /tailoring/cover-letters/"+created.ID+"/done", "")
	if done.DoneAt == nil {
		t.Errorf("POST done = %+v, want DoneAt set", done)
	}
	handlerstest.RequiresAuth(t, r, "GET /tailoring/cover-letters/"+created.ID+"/check", "POST /tailoring/cover-letters/"+created.ID+"/done")
}
