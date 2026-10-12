package applications_test

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
)

func Example_logRecruiterContacts() {
	ctx := context.Background()
	svc := applications.NewService(applicationstest.NewFakeStore())

	_, _ = svc.LogRecruiterContact(ctx, "user-1", dto.LogRecruiterContactInput{
		JobID:       "job-1",
		ContactedOn: "2026-10-12",
		Channel:     dto.ChannelLinkedInMessage,
		Outcome:     dto.OutcomeNoReply,
	})
	_, _ = svc.LogRecruiterContact(ctx, "user-1", dto.LogRecruiterContactInput{
		JobID:       "job-1",
		ContactedOn: "2026-10-14",
		Channel:     dto.ChannelCall,
		Outcome:     dto.OutcomeSpoke,
	})

	history, _ := svc.ListRecruiterContacts(ctx, "user-1", dto.RecruiterContactsQuery{JobID: "job-1"})
	_ = history

	summaries, _ := svc.RecruiterContactsForJobs(ctx, "user-1", dto.RecruiterContactsForJobsQuery{JobIDs: "job-1,job-2"})
	_ = summaries["job-1"].BestOutcome.Rank()
}
