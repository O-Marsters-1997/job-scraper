package jobsearchtest

import (
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func runPostedContract(t *testing.T, newStore func(t *testing.T) (Store, string)) {
	t.Helper()

	t.Run("a later save fills a missing posted date and a missing date never overwrites it", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := t.Context()
		posted := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
		job := dto.Job{Title: "Engineer", URL: "https://example.com/jobs/dates-1"}
		saved, _, err := st.SaveCanonical(ctx, job)
		if err != nil {
			t.Fatalf("SaveCanonical(no date) = %v", err)
		}
		if got, err := st.GetJob(ctx, saved.ID, missingID); err != nil || got.PostedAt != nil {
			t.Fatalf("GetJob(...) = %+v, %v, want a nil PostedAt", got, err)
		}

		job.PostedAt = &posted
		if _, status, err := st.SaveCanonical(ctx, job); err != nil || status != "unchanged" {
			t.Fatalf("SaveCanonical(dated) = %q, %v, want unchanged", status, err)
		}
		job.PostedAt = nil
		job.Title = "Senior Engineer"
		if _, status, err := st.SaveCanonical(ctx, job); err != nil || status != "changed" {
			t.Fatalf("SaveCanonical(changed, no date) = %q, %v, want changed", status, err)
		}
		got, err := st.GetJob(ctx, saved.ID, missingID)
		if err != nil || got.PostedAt == nil || !got.PostedAt.Equal(posted) || got.RepostedAt != nil {
			t.Fatalf("GetJob(...) = %+v, %v, want PostedAt %v and no RepostedAt", got, err, posted)
		}
	})

	t.Run("an earlier date wins and a date over two days later is a repost", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := t.Context()
		posted := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
		earlier := posted.AddDate(0, 0, -3)
		withinTwoDays := posted.AddDate(0, 0, 1)
		later := posted.AddDate(0, 0, 5)
		job := dto.Job{Title: "Engineer", URL: "https://example.com/jobs/dates-2", PostedAt: &posted}
		saved, _, err := st.SaveCanonical(ctx, job)
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		steps := []struct {
			name       string
			incoming   time.Time
			wantPosted time.Time
			wantRepost *time.Time
		}{
			{"a date within two days changes nothing", withinTwoDays, posted, nil},
			{"a later date sets the repost", later, posted, &later},
			{"an earlier date lowers the posted date", earlier, earlier, &later},
		}
		for _, step := range steps {
			job.PostedAt = &step.incoming
			if _, _, err := st.SaveCanonical(ctx, job); err != nil {
				t.Fatalf("%s: SaveCanonical(...) = %v", step.name, err)
			}
			got, err := st.GetJob(ctx, saved.ID, missingID)
			if err != nil || got.PostedAt == nil || !got.PostedAt.Equal(step.wantPosted) {
				t.Fatalf("%s: GetJob(...) = %+v, %v, want PostedAt %v", step.name, got, err, step.wantPosted)
			}
			if (got.RepostedAt == nil) != (step.wantRepost == nil) || (step.wantRepost != nil && !got.RepostedAt.Equal(*step.wantRepost)) {
				t.Fatalf("%s: RepostedAt = %v, want %v", step.name, got.RepostedAt, step.wantRepost)
			}
		}
	})

	t.Run("a job first seen after a completed poll carries that poll as its lower bound", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "range-co", Name: "Range Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		board, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", "range-co")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 60); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", "range-co", "test", ""); err != nil {
			t.Fatalf("VerifyCompanyBoard(...) = %v", err)
		}
		save := func(posting string) dto.Job {
			t.Helper()
			job, _, err := st.SaveCanonical(ctx, dto.Job{
				Title: "Role " + posting, URL: "https://boards.greenhouse.io/range-co/jobs/" + posting, Source: "greenhouse",
				CompanySlug: "range-co", CompanyID: c.ID, BoardID: board.ID, ProviderPostingID: posting,
			})
			if err != nil {
				t.Fatalf("SaveCanonical(%s) = %v", posting, err)
			}
			return job
		}

		first := save("1")
		poll, err := st.ClaimBoard(ctx, board.ID, false)
		if err != nil {
			t.Fatalf("ClaimBoard(...) = %v", err)
		}
		before := time.Now().Add(-time.Minute)
		if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: poll, Complete: true, Jobs: []dto.Job{first}, Reported: 1}); err != nil {
			t.Fatalf("CompleteBoard(...) = %v", err)
		}
		second := save("2")

		gotFirst, err := st.GetJob(ctx, first.ID, missingID)
		if err != nil || gotFirst.PostedAfter != nil {
			t.Fatalf("GetJob(first) = %+v, %v, want a nil PostedAfter", gotFirst, err)
		}
		gotSecond, err := st.GetJob(ctx, second.ID, missingID)
		if err != nil || gotSecond.PostedAfter == nil || gotSecond.PostedAfter.Before(before) {
			t.Fatalf("GetJob(second) = %+v, %v, want PostedAfter at the completed poll", gotSecond, err)
		}
	})
}
