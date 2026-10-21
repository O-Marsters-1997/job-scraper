package store_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func TestIngestJobsMergesWellfoundCardIntoATSJob(t *testing.T) {
	const (
		greenhouseURL = "https://boards.greenhouse.io/assemblyai/jobs/4855305"
		wellfoundURL  = "https://wellfound.com/jobs/4855305-software-engineer-agent-experience"
	)
	t.Parallel()
	st, pool, _ := newUserStore(t)
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "assemblyai", Name: "AssemblyAI"})
	board, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "assemblyai")
	if err != nil {
		t.Fatalf("UpsertCandidateBoard() err = %v", err)
	}
	svc := jobsearch.NewService(st, queuetest.NewRecorder(), jobsearchtest.NewNoopScoring())

	jobs := []dto.Job{
		{
			Title: "Software Engineer, Agent Experience", Location: "London", URL: greenhouseURL, Source: "greenhouse",
			CompanySlug: "assemblyai", CompanyID: company.ID, BoardID: board.ID, ProviderPostingID: "4855305",
		},
		{
			Title: "Software Engineer, Agent Experience", Location: "Boston, London, Remote", URL: wellfoundURL, Source: "wellfound",
			CompanySlug: "assemblyai", SalaryRaw: "$160k – $210k",
		},
	}
	results, err := svc.IngestJobs(t.Context(), jobs)
	if err != nil {
		t.Fatalf("IngestJobs() err = %v", err)
	}
	if results[0].Status != "new" || results[1].Status != "merged" || results[1].JobID != results[0].JobID {
		t.Fatalf("IngestJobs() = %+v, want a new Job then the Wellfound card merged into it", results)
	}
	if n := jobCount(t, pool); n != 1 {
		t.Errorf("jobs = %d, want 1", n)
	}
	if diff := cmp.Diff([]string{greenhouseURL, wellfoundURL}, listingURLs(t, pool, results[0].JobID)); diff != "" {
		t.Errorf("job_urls (-want +got):\n%s", diff)
	}
}
