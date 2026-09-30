package cvtemplates_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func newTestRouter(gc cvtemplates.DocsClient, st cvtemplates.Store) chi.Router {
	m := cvtemplates.Build(cvtemplates.Deps{Store: st, DocsClient: gc})
	r := chi.NewRouter()
	m.Routes(r)
	return r
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter(identitytest.NewDocsClient(), cvtemplatestest.NewFakeStore())

	handlerstest.RequiresAuth(t, r,
		"GET /cv-templates/",
		"GET /cv-templates/{docId}/{tabId}/pdf",
		"POST /tracked-docs/",
		"DELETE /tracked-docs/{id}",
		"POST /tracked-docs/{docId}/tabs/{tabId}/hide",
		"POST /tracked-docs/{docId}/tabs/{tabId}/show",
	)
	handlerstest.RejectsMalformedBody(t, r,
		"POST /tracked-docs/",
		"POST /tracked-docs/{docId}/tabs/{tabId}/hide",
		"POST /tracked-docs/{docId}/tabs/{tabId}/show",
	)
	handlerstest.RejectsBadPathID(t, r, "DELETE /tracked-docs/{id}")
}

func TestExportCV(t *testing.T) {
	r := newTestRouter(identitytest.NewDocsClient(), cvtemplatestest.NewFakeStore())

	w := handlerstest.Serve(t, r, "GET /cv-templates/docA/t1/pdf", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
	if got := w.Body.String(); got != "pdf-bytes" {
		t.Errorf("body = %q, want %q", got, "pdf-bytes")
	}
}

func TestTrackedDocRoutesHappyPaths(t *testing.T) {
	t.Run("list CVs", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		cvtemplatestest.Track(t, st, userID, "docA")
		r := newTestRouter(docsWith("docA"), st)

		got := handlerstest.Do[[]cvtemplates.CV](t, r, http.StatusOK, "GET /cv-templates/", "")

		if len(got) != 1 || got[0].DocID != "docA" || got[0].TabID != "t1" {
			t.Errorf("GET /cv-templates/ = %+v, want the docA t1 CV", got)
		}
	})

	t.Run("add tracked doc", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		r := newTestRouter(docsWith("doc-1"), st)

		handlerstest.Do[struct{}](t, r, http.StatusNoContent, "POST /tracked-docs/", `{"url":"https://docs.google.com/document/d/doc-1/edit"}`)

		docs, err := st.ListTrackedDocs(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 || docs[0].DocID != "doc-1" {
			t.Errorf("ListTrackedDocs = %+v, want doc-1", docs)
		}
	})

	t.Run("hide tab", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		id := seedTab(t, st, "doc-1", "t1", true)
		r := newTestRouter(identitytest.NewDocsClient(), st)

		handlerstest.Do[struct{}](t, r, http.StatusNoContent, "POST /tracked-docs/doc-1/tabs/t1/hide", "")

		if tabVisible(t, st, id, "t1") {
			t.Error("tab t1 visible after hide, want hidden")
		}
	})

	t.Run("show tab", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		id := seedTab(t, st, "doc-1", "t1", false)
		r := newTestRouter(identitytest.NewDocsClient(), st)

		handlerstest.Do[struct{}](t, r, http.StatusNoContent, "POST /tracked-docs/doc-1/tabs/t1/show", "")

		if !tabVisible(t, st, id, "t1") {
			t.Error("tab t1 hidden after show, want visible")
		}
	})

	t.Run("remove tracked doc", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		cvtemplatestest.Track(t, st, userID, "doc-1")
		r := newTestRouter(identitytest.NewDocsClient(), st)

		handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /tracked-docs/doc-1", "")

		docs, err := st.ListTrackedDocs(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 0 {
			t.Errorf("ListTrackedDocs = %+v, want none", docs)
		}
	})
}
