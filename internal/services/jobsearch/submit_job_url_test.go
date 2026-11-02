package jobsearch_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func TestSubmitJobURL(t *testing.T) {
	t.Parallel()
	const greenhouseURL = "https://boards.greenhouse.io/acme/jobs/4242"

	t.Run("a new ATS url tracks the company, publishes a verify and ingests nothing", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, st := newCompanyService(q)
		got, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: greenhouseURL})
		if err != nil || got.Status != dto.SubmissionQueued || got.JobID != "" {
			t.Fatalf("SubmitJobURL(...) = %+v, %v, want queued", got, err)
		}
		if len(q.tasks) != 1 || q.tasks[0].Kind != queue.BoardVerifyTask || !q.tasks[0].Urgent {
			t.Errorf("published = %+v, want one urgent verify", q.tasks)
		}
		tracked, err := st.ListTrackedCompaniesForUser(t.Context(), userID)
		if err != nil || len(tracked) != 1 {
			t.Errorf("ListTrackedCompaniesForUser(...) = %v, %v, want 1 tracked company", tracked, err)
		}
	})

	t.Run("a verified board publishes a board check", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, st := newCompanyService(q)
		verified, _ := seedBoards(t, st)
		if _, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: greenhouseURL}); err != nil {
			t.Fatalf("SubmitJobURL(...) err = %v", err)
		}
		if len(q.tasks) != 1 || q.tasks[0].Kind != queue.BoardCheckTask || q.tasks[0].BoardID != verified.ID {
			t.Errorf("published = %+v, want one board check for %s", q.tasks, verified.ID)
		}
	})

	t.Run("a held job returns held with its ID and publishes nothing", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, _ := newCompanyService(q)
		added, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: greenhouseURL, Title: "Engineer"})
		if err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		q.tasks = nil
		got, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: greenhouseURL})
		if err != nil || got.Status != dto.SubmissionHeld || got.JobID != added.Job.ID {
			t.Errorf("SubmitJobURL(...) = %+v, %v, want held %s", got, err, added.Job.ID)
		}
		if len(q.tasks) != 0 {
			t.Errorf("published = %+v, want nothing", q.tasks)
		}
	})

	t.Run("a linkedin or wis url queues one detail fetch on its own source", func(t *testing.T) {
		t.Parallel()
		tests := []struct{ name, in, source, want string }{
			{"linkedin slug", "https://www.linkedin.com/jobs/view/senior-engineer-at-acme-3901234567?trk=x", "linkedin", "https://www.linkedin.com/jobs/view/3901234567"},
			{"linkedin currentJobId", "https://www.linkedin.com/jobs/search/?currentJobId=3901234567", "linkedin", "https://www.linkedin.com/jobs/view/3901234567"},
			{"wis details", "https://workinstartups.com/details/777", "wis", "https://workinstartups.com/details/777"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				q := &recordingQueue{}
				svc, _ := newCompanyService(q)
				got, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: tt.in})
				if err != nil || got.Status != dto.SubmissionQueued {
					t.Fatalf("SubmitJobURL(%q) = %+v, %v, want queued", tt.in, got, err)
				}
				want := []dto.QueuedJob{{URL: tt.want, Card: dto.Job{URL: tt.want, Source: tt.source}}}
				if diff := cmp.Diff(want, q.jobs); diff != "" {
					t.Errorf("enqueued jobs (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("a linkedin url whose canonical form is held returns held and publishes nothing", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, _ := newCompanyService(q)
		added, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: "https://www.linkedin.com/jobs/view/3901234567", Title: "Engineer", Company: "Acme"})
		if err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		got, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: "https://www.linkedin.com/jobs/view/senior-engineer-3901234567?trk=x"})
		if err != nil || got.Status != dto.SubmissionHeld || got.JobID != added.Job.ID {
			t.Errorf("SubmitJobURL(...) = %+v, %v, want held %s", got, err, added.Job.ID)
		}
		if len(q.jobs) != 0 {
			t.Errorf("enqueued = %+v, want nothing", q.jobs)
		}
	})

	t.Run("an unsupported url is unprocessable, names the host and publishes nothing", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, _ := newCompanyService(q)
		_, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: "https://uk.indeed.com/viewjob?jk=1"})
		if !apperr.IsKind(err, apperr.KindUnprocessable) {
			t.Fatalf("SubmitJobURL(indeed) err = %v, want unprocessable", err)
		}
		if msg := apperr.MessageFor(err); !strings.Contains(msg, "uk.indeed.com") || !strings.Contains(msg, "Add application") {
			t.Errorf("message = %q, want host and Add application", msg)
		}
		if len(q.tasks) != 0 {
			t.Errorf("published = %+v, want nothing", q.tasks)
		}
	})

	t.Run("a malformed url is invalid", func(t *testing.T) {
		t.Parallel()
		svc, _ := newCompanyService(&recordingQueue{})
		_, err := svc.SubmitJobURL(t.Context(), userID, dto.SubmitJobURLInput{URL: "not a url"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("SubmitJobURL(bad) err = %v, want invalid", err)
		}
	})
}
