package applications_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
)

const testUserID = "route-test-user"

func newTestRouter() chi.Router {
	m := applications.Build(applications.Deps{Store: applicationstest.NewFakeStore()})
	r := chi.NewRouter()
	m.Routes(r)
	return r
}

func authedRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(handlers.WithSession(req.Context(), dto.Session{UserID: testUserID}))
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter()

	handlerstest.RequiresAuth(t, r,
		"GET /application-statuses/",
		"POST /application-statuses/",
		"PATCH /application-statuses/{id}",
		"DELETE /application-statuses/{id}",
		"GET /applications/",
		"POST /applications/",
		"PATCH /applications/{id}",
		"DELETE /applications/{id}",
		"GET /applications/for-jobs",
	)
	handlerstest.RejectsMalformedBody(t, r,
		"POST /application-statuses/", "PATCH /application-statuses/{id}",
		"POST /applications/", "PATCH /applications/{id}",
	)
	handlerstest.RejectsBadPathID(t, r, "PATCH /applications/{id}")
}

func TestRoutesHappyPaths(t *testing.T) {
	r := newTestRouter()
	var statusID, appID string

	t.Run("create status", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodPost, "/application-statuses/", `{"name":"Applied","colour":"#6366f1"}`))
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var got dto.ApplicationStatus
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Name != "Applied" {
			t.Fatalf("name = %q, want Applied", got.Name)
		}
		statusID = got.ID
	})

	t.Run("list statuses", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodGet, "/application-statuses/", ""))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var got []dto.ApplicationStatus
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d statuses, want 1", len(got))
		}
	})

	t.Run("update status", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodPatch, "/application-statuses/"+statusID, `{"name":"Applied!","colour":"#6366f1"}`))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
	})

	t.Run("create application", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodPost, "/applications/", `{"job_id":"job-1","status_id":"`+statusID+`"}`))
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var got dto.Application
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.JobID != "job-1" {
			t.Fatalf("job_id = %q, want job-1", got.JobID)
		}
		appID = got.ID
	})

	t.Run("list applications", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodGet, "/applications/", ""))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var got []dto.ApplicationWithDetails
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d applications, want 1", len(got))
		}
	})

	t.Run("update application", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodPatch, "/applications/"+appID, `{"notes":"followed up"}`))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var got dto.Application
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Notes != "followed up" {
			t.Fatalf("notes = %q, want followed up", got.Notes)
		}
	})

	t.Run("applications for jobs", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodGet, "/applications/for-jobs?job_ids=job-1", ""))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var got map[string]dto.JobApplicationSummary
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if _, ok := got["job-1"]; !ok {
			t.Fatalf("got = %+v, want job-1 present", got)
		}
	})

	t.Run("delete application", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodDelete, "/applications/"+appID, ""))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
	})

	t.Run("delete status", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authedRequest(t, http.MethodDelete, "/application-statuses/"+statusID, ""))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
	})
}
