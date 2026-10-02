package cvtailor_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func newRouter(deps cvtailor.Deps) chi.Router {
	if deps.Store == nil {
		deps.Store = cvtailortest.NewFakeStore()
	}
	r := chi.NewRouter()
	cvtailor.Build(deps).Routes(r)
	return r
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newRouter(cvtailor.Deps{})
	handlerstest.RequiresAuth(t, r,
		"GET /experience/",
		"POST /experience/positions",
		"PUT /experience/positions/order",
		"PATCH /experience/positions/{id}",
		"DELETE /experience/positions/{id}",
		"POST /experience/positions/{id}/achievements",
		"PUT /experience/positions/{id}/achievements/order",
		"PATCH /experience/achievements/{id}",
		"DELETE /experience/achievements/{id}",
		"POST /experience/import/preview",
		"POST /experience/import",
		"GET /tailoring/cvs/{docId}/{tabId}/headings",
		"PUT /tailoring/cvs/{docId}/{tabId}/headings",
		"GET /tailoring/jobs/{jobId}/suggestions",
		"POST /tailoring/jobs/{jobId}/achievements/{achievementId}/explain",
		"POST /tailoring/drafts",
		"GET /tailoring/drafts/{id}",
		"GET /tailoring/drafts/{id}/layout",
		"POST /tailoring/drafts/{id}/slots/{slotId}/suggest",
		"GET /tailoring/drafts/{id}/pdf",
		"POST /tailoring/drafts/{id}/keep",
		"POST /tailoring/drafts/{id}/discard",
		"GET /tailoring/jobs/{jobId}/drafts",
	)
	handlerstest.RejectsMalformedBody(t, r,
		"POST /experience/positions",
		"PUT /experience/positions/order",
		"PATCH /experience/positions/{id}",
		"POST /experience/positions/{id}/achievements",
		"PATCH /experience/achievements/{id}",
		"POST /experience/import/preview",
		"POST /experience/import",
		"PUT /tailoring/cvs/{docId}/{tabId}/headings",
		"POST /tailoring/drafts",
		"POST /tailoring/drafts/{id}/slots/{slotId}/suggest",
	)
	handlerstest.RejectsBadPathID(t, r,
		"DELETE /experience/positions/{id}",
		"DELETE /experience/achievements/{id}",
	)
}

func TestExperienceJourney(t *testing.T) {
	r := newRouter(cvtailor.Deps{})

	p := handlerstest.Do[dto.Position](t, r, http.StatusCreated, "POST /experience/positions", `{"employer":"Acme","title":"Engineer","startDate":"2020-01-01"}`)
	a := handlerstest.Do[dto.Achievement](t, r, http.StatusCreated, "POST /experience/positions/"+p.ID+"/achievements", `{"text":"cut latency"}`)
	handlerstest.Do[dto.Achievement](t, r, http.StatusOK, "PATCH /experience/achievements/"+a.ID, `{"text":"cut p99 latency"}`)
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "PUT /experience/positions/"+p.ID+"/achievements/order", `{"ids":["`+a.ID+`"]}`)
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "PUT /experience/positions/order", `{"ids":["`+p.ID+`"]}`)

	got := handlerstest.Do[[]dto.Position](t, r, http.StatusOK, "GET /experience/", "")

	want := []dto.Position{{
		ID: p.ID, Employer: "Acme", Title: "Engineer", StartDate: ptr("2020-01-01"),
		Achievements: []dto.Achievement{{ID: a.ID, PositionID: p.ID, Text: "cut p99 latency"}},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("GET /experience/ (-want +got):\n%s", diff)
	}
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /experience/positions/"+p.ID, "")
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "DELETE /experience/achievements/"+a.ID, "")
}

func TestImportRoutes(t *testing.T) {
	r := newRouter(cvtailor.Deps{Docs: cvtailortest.Docs{TabJSON: tabJSON(t, head("Engineer, Acme"), bullet("shipped"))}})

	preview := handlerstest.Serve(t, r, "POST /experience/import/preview", `{"docId":"d","tabId":"t"}`)
	if preview.Code != http.StatusOK {
		t.Fatalf("POST preview = %d: %s", preview.Code, preview.Body)
	}
	handlerstest.Do[struct{}](t, r, http.StatusCreated, "POST /experience/import", preview.Body.String())

	got := handlerstest.Do[[]dto.Position](t, r, http.StatusOK, "GET /experience/", "")

	if len(got) != 1 || got[0].Employer != "Acme" || len(got[0].Achievements) != 1 {
		t.Fatalf("GET /experience/ after import = %+v, want one Acme position with one achievement", got)
	}
}

func TestDraftRoutesQueueAndReadADraft(t *testing.T) {
	r := newRouter(cvtailor.Deps{})
	p := handlerstest.Do[dto.Position](t, r, http.StatusCreated, "POST /experience/positions", `{"employer":"Acme","title":"Engineer"}`)
	a := handlerstest.Do[dto.Achievement](t, r, http.StatusCreated, "POST /experience/positions/"+p.ID+"/achievements", `{"text":"cut latency"}`)
	handlerstest.Do[struct{}](t, r, http.StatusOK, "PUT /tailoring/cvs/d/t/headings", `{"mappings":[{"headingText":"Acme","positionId":"`+p.ID+`"}]}`)

	ref := handlerstest.Do[dto.DraftRef](t, r, http.StatusAccepted, "POST /tailoring/drafts", `{"jobId":"job-1","docId":"d","tabId":"t","achievementIds":["`+a.ID+`"]}`)

	got := handlerstest.Do[dto.Draft](t, r, http.StatusOK, "GET /tailoring/drafts/"+ref.ID, "")
	if got.ID != ref.ID || got.Status != "pending" || got.DraftDocURL != nil {
		t.Errorf("GET draft = %+v, want the pending Draft", got)
	}
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "GET /tailoring/drafts/missing", "")
}

func TestReviewRoutes(t *testing.T) {
	t.Run("streams the PDF of a ready Draft", func(t *testing.T) {
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		r := newRouter(cvtailor.Deps{Store: e.store, Drive: cvtailortest.ExportsPDF(e.drive, "%PDF-fake")})

		w := handlerstest.Serve(t, r, "GET /tailoring/drafts/"+id+"/pdf", "")

		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/pdf" || w.Body.String() != "%PDF-fake" {
			t.Errorf("GET pdf = %d %q %q, want the streamed PDF", w.Code, w.Header().Get("Content-Type"), w.Body)
		}
	})

	t.Run("keeps, lists and discards a Job's Drafts", func(t *testing.T) {
		e := newDraftEnv(t)
		r := newRouter(cvtailor.Deps{Store: e.store, Drive: e.drive})
		ids := e.readyDrafts(t, 2)

		handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+ids[0]+"/keep", "")
		handlerstest.Do[struct{}](t, r, http.StatusConflict, "POST /tailoring/drafts/"+ids[1]+"/keep", "")
		e.run(t, tick{})

		list := handlerstest.Do[[]dto.Draft](t, r, http.StatusOK, "GET /tailoring/jobs/"+jobID+"/drafts", "")
		var gotIDs []string
		for _, d := range list {
			gotIDs = append(gotIDs, d.ID)
		}
		if diff := cmp.Diff(ids, gotIDs, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
			t.Errorf("listed Draft ids (-want +got):\n%s", diff)
		}

		handlerstest.Do[dto.Draft](t, r, http.StatusOK, "POST /tailoring/drafts/"+ids[0]+"/discard", "")
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
