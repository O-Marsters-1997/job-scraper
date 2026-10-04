package applications_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
)

func newTestRouter() chi.Router {
	m := applications.Build(applications.Deps{Store: applicationstest.NewFakeStore()})
	r := chi.NewRouter()
	m.Routes(r)
	return r
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
		"PUT /applications/{id}/chase",
		"DELETE /applications/{id}/chase",
		"GET /applications/for-jobs",
	)
	handlerstest.RejectsMalformedBody(t, r,
		"POST /application-statuses/", "PATCH /application-statuses/{id}",
		"POST /applications/", "PATCH /applications/{id}", "PUT /applications/{id}/chase",
	)
	handlerstest.RejectsBadPathID(t, r, "PATCH /applications/{id}")
}

func TestRoutesHappyPaths(t *testing.T) {
	r := newTestRouter()

	status := handlerstest.Do[dto.ApplicationStatus](t, r, http.StatusCreated,
		"POST /application-statuses/", `{"name":"Applied","colour":"#6366f1"}`)
	if status.Name != "Applied" {
		t.Fatalf("created status name = %q, want Applied", status.Name)
	}

	statuses := handlerstest.Do[[]dto.ApplicationStatus](t, r, http.StatusOK, "GET /application-statuses/", "")
	if len(statuses) != 1 {
		t.Fatalf("got %d statuses, want 1", len(statuses))
	}

	handlerstest.Do[struct{}](t, r, http.StatusOK,
		"PATCH /application-statuses/"+status.ID, `{"name":"Applied!","colour":"#6366f1"}`)

	app := handlerstest.Do[dto.Application](t, r, http.StatusCreated,
		"POST /applications/", `{"job_id":"job-1","status_id":"`+status.ID+`"}`)
	if app.JobID != "job-1" {
		t.Fatalf("created job_id = %q, want job-1", app.JobID)
	}

	apps := handlerstest.Do[[]dto.ApplicationWithDetails](t, r, http.StatusOK, "GET /applications/", "")
	if len(apps) != 1 {
		t.Fatalf("got %d applications, want 1", len(apps))
	}

	updated := handlerstest.Do[dto.Application](t, r, http.StatusOK,
		"PATCH /applications/"+app.ID, `{"notes":"followed up"}`)
	if updated.Notes != "followed up" {
		t.Fatalf("updated notes = %q, want followed up", updated.Notes)
	}

	chased := handlerstest.Do[dto.Application](t, r, http.StatusOK,
		"PUT /applications/"+app.ID+"/chase", `{"chase_by":"2026-10-20"}`)
	if chased.ChaseBy == nil || chased.ChaseBy.Format("2006-01-02") != "2026-10-20" {
		t.Fatalf("chased ChaseBy = %v, want 2026-10-20", chased.ChaseBy)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest,
		"PUT /applications/"+app.ID+"/chase", `{"chase_by":"soon"}`)
	if got := handlerstest.Do[[]dto.ApplicationWithDetails](t, r, http.StatusOK, "GET /applications/?chase=true", ""); len(got) != 1 {
		t.Fatalf("chase=true listed %d applications, want 1", len(got))
	}
	for range 2 {
		handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /applications/"+app.ID+"/chase", "")
	}
	if got := handlerstest.Do[[]dto.ApplicationWithDetails](t, r, http.StatusOK, "GET /applications/?chase=true", ""); len(got) != 0 {
		t.Fatalf("chase=true after clear listed %d applications, want 0", len(got))
	}

	summaries := handlerstest.Do[map[string]dto.JobApplicationSummary](t, r, http.StatusOK,
		"GET /applications/for-jobs?job_ids=job-1", "")
	if _, ok := summaries["job-1"]; !ok {
		t.Fatalf("for-jobs = %+v, want job-1 present", summaries)
	}

	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /applications/"+app.ID, "")
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /application-statuses/"+status.ID, "")
}
