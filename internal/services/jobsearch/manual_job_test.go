package jobsearch_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func TestAddManualJob(t *testing.T) {
	t.Parallel()
	const greenhouseURL = "https://boards.greenhouse.io/acme/jobs/4242"

	t.Run("an ATS url saves the job on the candidate board, with its posting ID, and tracks the company", func(t *testing.T) {
		t.Parallel()
		svc, st := newCompanyService(&recordingQueue{})
		got, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: greenhouseURL, Title: "Engineer", Company: "Acme Ltd"})
		if err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		if got.Existing || got.Job.Source != "greenhouse" || got.Job.ProviderPostingID != "4242" || got.Job.BoardID == "" {
			t.Errorf("AddManualJob(...) = %+v, want a new greenhouse job with board and posting ID", got)
		}
		tracked, err := st.ListTrackedCompaniesForUser(t.Context(), userID)
		if err != nil || len(tracked) != 1 {
			t.Fatalf("ListTrackedCompaniesForUser(...) = %v, %v, want 1 tracked company", tracked, err)
		}
		company, err := st.GetCompany(t.Context(), got.Job.CompanyID)
		if err != nil || company.Name != "Acme Ltd" {
			t.Errorf("GetCompany(...) = %+v, %v, want name Acme Ltd", company, err)
		}
	})

	t.Run("a url with no ATS saves a job with source other", func(t *testing.T) {
		t.Parallel()
		svc, _ := newCompanyService(&recordingQueue{})
		got, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: "https://careers.example.com/role/9", Title: "Engineer", Company: "Example Co"})
		if err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		if got.Existing || got.Job.Source != "other" || got.Job.CompanySlug != "example" {
			t.Errorf("AddManualJob(...) = %+v, want a new other job for example", got)
		}
	})

	t.Run("a url already held returns the existing job unchanged", func(t *testing.T) {
		t.Parallel()
		for _, url := range []string{greenhouseURL, "https://careers.example.com/role/9"} {
			svc, _ := newCompanyService(&recordingQueue{})
			in := dto.AddManualJobInput{URL: url, Title: "Engineer", Company: "Acme", Description: "crawled"}
			first, err := svc.AddManualJob(t.Context(), userID, in)
			if err != nil {
				t.Fatalf("AddManualJob(first %s) err = %v", url, err)
			}
			in.Title, in.Description = "Renamed", "typed by hand"
			second, err := svc.AddManualJob(t.Context(), userID, in)
			if err != nil {
				t.Fatalf("AddManualJob(second %s) err = %v", url, err)
			}
			if !second.Existing || second.Job.ID != first.Job.ID || second.Job.Title != "Engineer" || second.Job.Description != "crawled" || second.Job.Source != first.Job.Source {
				t.Errorf("AddManualJob(second %s) = %+v, want existing %+v", url, second, first)
			}
		}
	})

	t.Run("a crawled job is returned for its posting even when the url differs", func(t *testing.T) {
		t.Parallel()
		svc, st := newCompanyService(&recordingQueue{})
		company := seedCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		board, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard(...) err = %v", err)
		}
		crawled, _, err := st.SaveCanonical(t.Context(), dto.Job{
			Title: "Crawled", URL: "https://job-boards.greenhouse.io/acme/jobs/4242", Source: "greenhouse",
			CompanySlug: "acme", BoardID: board.ID, ProviderPostingID: "4242", Description: "real",
		})
		if err != nil {
			t.Fatalf("SaveCanonical(...) err = %v", err)
		}
		got, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: greenhouseURL, Title: "Mine"})
		if err != nil || !got.Existing || got.Job.ID != crawled.ID || got.Job.Title != "Crawled" {
			t.Errorf("AddManualJob(...) = %+v, %v, want existing crawled job", got, err)
		}
	})

	t.Run("a new job on a verified board publishes an urgent manual board check", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, st := newCompanyService(q)
		verified, _ := seedBoards(t, st)
		if _, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: greenhouseURL, Title: "Engineer"}); err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		if len(q.tasks) != 1 || q.tasks[0].Kind != queue.BoardCheckTask || q.tasks[0].BoardID != verified.ID || !q.tasks[0].Manual || !q.tasks[0].Urgent {
			t.Errorf("published = %+v, want one urgent manual board check for %s", q.tasks, verified.ID)
		}
	})

	t.Run("a new job on an unverified board publishes an urgent polling verify", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, _ := newCompanyService(q)
		got, err := svc.AddManualJob(t.Context(), userID, dto.AddManualJobInput{URL: greenhouseURL, Title: "Engineer"})
		if err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		if len(q.tasks) != 1 {
			t.Fatalf("published = %+v, want one task", q.tasks)
		}
		if task := q.tasks[0]; task.Kind != queue.BoardVerifyTask || task.CompanyID != got.Job.CompanyID || task.BoardToken != "acme" || !task.Poll || !task.Urgent {
			t.Errorf("published = %+v, want urgent polling verify for acme", task)
		}
	})

	t.Run("a non-ATS url publishes nothing", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, _ := newCompanyService(q)
		in := dto.AddManualJobInput{URL: "https://careers.example.com/role/9", Title: "Engineer", Company: "Example"}
		if _, err := svc.AddManualJob(t.Context(), userID, in); err != nil {
			t.Fatalf("AddManualJob(...) err = %v", err)
		}
		if len(q.tasks) != 0 {
			t.Errorf("published = %+v, want nothing", q.tasks)
		}
	})

	t.Run("re-adding a held ATS url publishes nothing more", func(t *testing.T) {
		t.Parallel()
		q := &recordingQueue{}
		svc, _ := newCompanyService(q)
		in := dto.AddManualJobInput{URL: greenhouseURL, Title: "Engineer"}
		for range 2 {
			if _, err := svc.AddManualJob(t.Context(), userID, in); err != nil {
				t.Fatalf("AddManualJob(...) err = %v", err)
			}
		}
		if len(q.tasks) != 1 {
			t.Errorf("published = %+v, want only the first add's task", q.tasks)
		}
	})

	t.Run("rejects missing input", func(t *testing.T) {
		t.Parallel()
		for _, in := range []dto.AddManualJobInput{
			{Title: "Engineer", Company: "Acme"},
			{URL: "https://careers.example.com/role/9", Company: "Acme"},
			{URL: "https://careers.example.com/role/9", Title: "Engineer"},
			{URL: "ftp://careers.example.com/role/9", Title: "Engineer", Company: "Acme"},
		} {
			svc, _ := newCompanyService(&recordingQueue{})
			if _, err := svc.AddManualJob(t.Context(), userID, in); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("AddManualJob(%+v) err = %v, want kind %v", in, err, apperr.KindInvalid)
			}
		}
	})
}
