package jobsearch_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
)

func TestIngestJobsPublishesBoardDiscover(t *testing.T) {
	const ashbyURL = "https://jobs.ashbyhq.com/acme/123"

	tests := []struct {
		name     string
		verified bool
		jobs     []dto.Job
		want     []queue.Task
	}{
		{
			name: "unseen ATS apply url publishes one task",
			jobs: []dto.Job{{Title: "Eng", URL: ashbyURL, Source: "linkedin"}},
			want: []queue.Task{{Source: "ashby", Kind: queue.BoardDiscoverTask, BoardToken: "acme", Via: "linkedin"}},
		},
		{
			name: "same board twice in a batch publishes once",
			jobs: []dto.Job{
				{Title: "Eng", URL: ashbyURL, Source: "linkedin"},
				{Title: "PM", URL: "https://jobs.ashbyhq.com/acme/456", Source: "linkedin"},
			},
			want: []queue.Task{{Source: "ashby", Kind: queue.BoardDiscoverTask, BoardToken: "acme", Via: "linkedin"}},
		},
		{
			name:     "verified board publishes none",
			verified: true,
			jobs:     []dto.Job{{Title: "Eng", URL: ashbyURL, Source: "linkedin"}},
		},
		{
			name: "job from its own ATS publishes none",
			jobs: []dto.Job{{Title: "Eng", URL: ashbyURL, Source: "ashby"}},
		},
		{
			name: "apply url on another host publishes its board",
			jobs: []dto.Job{{Title: "Eng", URL: "https://app.welcometothejungle.com/jobs/x", ApplyURL: ashbyURL, Source: "wttj"}},
			want: []queue.Task{{Source: "ashby", Kind: queue.BoardDiscoverTask, BoardToken: "acme", Via: "wttj"}},
		},
		{
			name: "pinpoint apply url publishes its board",
			jobs: []dto.Job{{Title: "Eng", URL: "https://app.welcometothejungle.com/jobs/x", ApplyURL: "https://acme.pinpointhq.com/en/postings/ce6c9e5c-a2d3", Source: "wttj"}},
			want: []queue.Task{{Source: "pinpoint", Kind: queue.BoardDiscoverTask, BoardToken: "acme", Via: "wttj"}},
		},
		{
			name: "unresolved apply url publishes none",
			jobs: []dto.Job{{Title: "Eng", URL: "https://app.welcometothejungle.com/jobs/x", ApplyURL: "https://x.wd1.myworkdayjobs.com/j/1", Source: "wttj"}},
		},
		{
			name: "unresolved host publishes none",
			jobs: []dto.Job{{Title: "Eng", URL: "https://careers.example.com/1", Source: "linkedin"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := queuetest.NewRecorder()
			svc, st := newCompanyService(rec)
			if tt.verified {
				company := seedCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
				if _, err := st.UpsertCandidateBoard(t.Context(), company.ID, "ashby", "acme"); err != nil {
					t.Fatalf("UpsertCandidateBoard err = %v", err)
				}
				if _, err := st.VerifyCompanyBoard(t.Context(), company.ID, "ashby", "acme", "test", ""); err != nil {
					t.Fatalf("VerifyCompanyBoard err = %v", err)
				}
			}

			if _, err := svc.IngestJobs(t.Context(), tt.jobs); err != nil {
				t.Fatalf("IngestJobs err = %v", err)
			}

			var got []queue.Task
			for _, task := range rec.Tasks() {
				got = append(got, queue.Task{Source: task.Source, Kind: task.Kind, BoardToken: task.BoardToken, Via: task.Via})
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("published tasks (-want +got):\n%s", diff)
			}
		})
	}
}
