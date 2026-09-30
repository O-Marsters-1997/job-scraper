package sourcetargets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

type failingJobQueue struct {
	*queuetest.Recorder
	err error
}

func (q *failingJobQueue) EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error {
	if q.err != nil {
		return q.err
	}
	return q.Recorder.EnqueueJobs(ctx, jobs)
}

var _ sourcetargets.QueuePublisher = (*failingJobQueue)(nil)

var (
	captureTarget = dto.SourceTarget{ID: "target-1", UserID: userID, Source: "wis"}
	captureCard   = dto.Job{URL: "https://example.com/1", Title: "Engineer"}
)

func TestCapturePage(t *testing.T) {
	t.Run("retries after a queue failure", func(t *testing.T) {
		q := &failingJobQueue{Recorder: queuetest.NewRecorder(), err: errors.New("queue unavailable")}
		svc := serviceOver(jobsearchtest.NewFakeStore(), q)
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), captureTarget, []dto.Job{captureCard}, config); err == nil {
			t.Fatal("CapturePage() err = nil, want the queue failure")
		}

		q.err = nil
		if err := svc.Reconsider(t.Context(), config); err != nil {
			t.Fatalf("Reconsider() err = %v", err)
		}
		if got := len(q.Jobs()); got != 1 {
			t.Errorf("retry queued %d details, want 1", got)
		}
	})

	t.Run("retains a rejected card and reconsideration queues it once", func(t *testing.T) {
		svc, _, q := newService(t)
		card := captureCard
		card.Title = "Senior Engineer"
		first := dto.SearchConfig{UserID: userID, ExcludedTitleKeywords: []string{"senior"}, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), captureTarget, []dto.Job{card, card}, first); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}
		if got := len(q.Jobs()); got != 0 {
			t.Fatalf("rejected card queued %d details, want 0", got)
		}

		changed := dto.SearchConfig{UserID: userID, UpdatedAt: first.UpdatedAt.Add(time.Second)}
		for range 2 {
			if err := svc.Reconsider(t.Context(), changed); err != nil {
				t.Fatalf("Reconsider() err = %v", err)
			}
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != card.URL {
			t.Errorf("queued after reconsideration = %+v, want one job for %s", jobs, card.URL)
		}
	})
}
