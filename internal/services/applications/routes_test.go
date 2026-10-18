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
		"GET /recruiter-contacts/summaries",
		"GET /jobs/{id}/recruiter-contacts",
		"POST /jobs/{id}/recruiter-contacts",
		"GET /jobs/{id}/notes",
		"POST /jobs/{id}/notes", "PATCH /notes/{id}", "DELETE /notes/{id}",
		"GET /jobs/{id}/comments",
		"POST /jobs/{id}/comments", "PATCH /comments/{id}", "DELETE /comments/{id}",
	)
	handlerstest.RejectsMalformedBody(t, r,
		"POST /application-statuses/", "PATCH /application-statuses/{id}",
		"POST /applications/", "PATCH /applications/{id}", "PUT /applications/{id}/chase",
		"POST /jobs/{id}/recruiter-contacts", "POST /jobs/{id}/notes",
		"PATCH /notes/{id}", "POST /jobs/{id}/comments", "PATCH /comments/{id}",
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
		"PATCH /applications/"+app.ID, `{"salary_info":"60k"}`)
	if updated.SalaryInfo != "60k" {
		t.Fatalf("updated salary = %q, want 60k", updated.SalaryInfo)
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

	logged := handlerstest.Do[dto.RecruiterContact](t, r, http.StatusCreated, "POST /jobs/job-1/recruiter-contacts",
		`{"contacted_on":"2026-10-01","channel":"email","outcome":"spoke"}`)
	if logged.JobID != "job-1" {
		t.Fatalf("logged job_id = %q, want job-1", logged.JobID)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest, "POST /jobs/job-1/recruiter-contacts",
		`{"contacted_on":"2026-10-01","channel":"fax","outcome":"spoke"}`)
	if got := handlerstest.Do[[]dto.RecruiterContact](t, r, http.StatusOK, "GET /jobs/job-1/recruiter-contacts", ""); len(got) != 1 {
		t.Fatalf("got %d recruiter contacts, want 1", len(got))
	}
	sums := handlerstest.Do[map[string]dto.RecruiterContactSummary](t, r, http.StatusOK, "GET /recruiter-contacts/summaries?job_ids=job-1,job-2", "")
	if sums["job-1"].Count != 1 || sums["job-1"].BestOutcome != dto.OutcomeSpoke || len(sums) != 1 {
		t.Fatalf("summaries = %+v, want only job-1 with 1 spoke contact", sums)
	}

	note := handlerstest.Do[dto.JobNote](t, r, http.StatusCreated, "POST /jobs/job-1/notes", `{"body":"hello"}`)
	if note.JobID != "job-1" || note.Body != "hello" {
		t.Fatalf("created note = %+v, want job-1/hello", note)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest, "POST /jobs/job-1/notes", `{"body":""}`)
	if got := handlerstest.Do[[]dto.JobNote](t, r, http.StatusOK, "GET /jobs/job-1/notes", ""); len(got) != 1 {
		t.Fatalf("got %d notes, want 1", len(got))
	}
	edited := handlerstest.Do[dto.JobNote](t, r, http.StatusOK, "PATCH /notes/"+note.ID, `{"body":"changed"}`)
	if edited.Body != "changed" {
		t.Fatalf("edited note body = %q, want changed", edited.Body)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest, "PATCH /notes/"+note.ID, `{"body":""}`)
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /notes/"+note.ID, "")
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "DELETE /notes/"+note.ID, "")

	comment := handlerstest.Do[dto.JobComment](t, r, http.StatusCreated, "POST /jobs/job-1/comments",
		`{"body":"ask","quote":{"exact":"Go","prefix":"we use ","suffix":" daily"}}`)
	if comment.JobID != "job-1" || comment.Quote.Exact != "Go" {
		t.Fatalf("created comment = %+v, want job-1 quoting Go", comment)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest, "POST /jobs/job-1/comments", `{"body":"ask"}`)
	if got := handlerstest.Do[[]dto.JobComment](t, r, http.StatusOK, "GET /jobs/job-1/comments", ""); len(got) != 1 {
		t.Fatalf("got %d comments, want 1", len(got))
	}
	handlerstest.Do[dto.JobComment](t, r, http.StatusOK, "PATCH /comments/"+comment.ID, `{"body":"changed"}`)
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /comments/"+comment.ID, "")
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "DELETE /comments/"+comment.ID, "")

	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /applications/"+app.ID, "")
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /application-statuses/"+status.ID, "")
}
