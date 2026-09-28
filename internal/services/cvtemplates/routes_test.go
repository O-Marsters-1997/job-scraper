package cvtemplates_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs/trackeddocstest"
)

const testUserID = "route-test-user"

func newTestRouter(gc cvtemplates.DocsClient, trackedDocs *trackeddocstest.FakeStore) chi.Router {
	m := cvtemplates.Build(cvtemplates.Deps{
		CV:          cvtemplatestest.NewFakeStore(),
		TrackedDocs: trackedDocs,
		DocsClient:  gc,
	})
	r := chi.NewRouter()
	m.Routes(r)
	return r
}

func authedRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(handlers.WithSession(req.Context(), dto.Session{UserID: testUserID}))
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter(identitytest.NewDocsClient(), trackeddocstest.NewFakeStore())

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
	t.Run("streams the PDF on success", func(t *testing.T) {
		r := newTestRouter(identitytest.NewDocsClient(), trackeddocstest.NewFakeStore())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(http.MethodGet, "/cv-templates/docA/t1/pdf", ""))

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
			t.Errorf("Content-Type = %q, want application/pdf", ct)
		}
		if w.Body.Len() == 0 {
			t.Error("expected a non-empty PDF body")
		}
	})

	t.Run("maps an upstream failure to 502", func(t *testing.T) {
		gc := failingExport{DocsClient: identitytest.NewDocsClient(), err: errors.New("google is down")}
		r := newTestRouter(gc, trackeddocstest.NewFakeStore())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(http.MethodGet, "/cv-templates/docA/t1/pdf", ""))

		if w.Code != http.StatusBadGateway {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
	})
}

func TestTrackedDocRoutesHappyPaths(t *testing.T) {
	trackedDocs := trackeddocstest.NewFakeStore()
	gc := identitytest.NewDocsClient().WithDoc("doc-1", nil, google.FileMeta{Title: "My CV"})
	r := newTestRouter(gc, trackedDocs)

	t.Run("add tracked doc", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(http.MethodPost, "/tracked-docs/", `{"url":"https://docs.google.com/document/d/doc-1/edit"}`))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
	})

	t.Run("hide tab", func(t *testing.T) {
		trackedDocs.SeedTab(testUserID, "doc-1", "t1", true)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(http.MethodPost, "/tracked-docs/doc-1/tabs/t1/hide", ""))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		if trackedDocs.Visible(testUserID, "doc-1", "t1") {
			t.Error("tab should be hidden")
		}
	})

	t.Run("show tab", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(http.MethodPost, "/tracked-docs/doc-1/tabs/t1/show", ""))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		if !trackedDocs.Visible(testUserID, "doc-1", "t1") {
			t.Error("tab should be visible")
		}
	})

	t.Run("remove tracked doc", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(http.MethodDelete, "/tracked-docs/doc-1", ""))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
	})
}
