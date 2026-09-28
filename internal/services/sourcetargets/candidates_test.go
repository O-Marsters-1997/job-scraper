package sourcetargets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
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

func TestCaptureRetriesAfterQueueFailure(t *testing.T) {
	ctx := context.Background()
	q := &failingJobQueue{Recorder: queuetest.NewRecorder(), err: errors.New("queue unavailable")}
	svc := newService(newFakeStore(), q)
	target := dto.SourceTarget{ID: "target-1", UserID: "user-1", Source: "wis"}
	card := dto.Job{URL: "https://example.com/1", Title: "Engineer"}
	config := dto.SearchConfig{UserID: target.UserID, UpdatedAt: time.Now().UTC()}
	if err := svc.CapturePage(ctx, target, []dto.Job{card}, config); err == nil {
		t.Fatal("CapturePage(...) err = nil, want queue failure")
	}

	q.err = nil
	if err := svc.Reconsider(ctx, config); err != nil {
		t.Fatalf("Reconsider(...) = %v", err)
	}
	if got := len(q.Jobs()); got != 1 {
		t.Fatalf("retry queued %d details, want 1", got)
	}
}

func TestCaptureRetainsRejectedCardAndReconsiderationQueuesOnce(t *testing.T) {
	ctx := context.Background()
	q := queuetest.NewRecorder()
	svc := newService(newFakeStore(), q)
	target := dto.SourceTarget{ID: "target-1", UserID: "user-1", Source: "wis"}
	card := dto.Job{URL: "https://example.com/1", Title: "Senior Engineer"}
	first := dto.SearchConfig{UserID: target.UserID, ExcludedTitleKeywords: []string{"senior"}, UpdatedAt: time.Now().UTC()}
	if err := svc.CapturePage(ctx, target, []dto.Job{card, card}, first); err != nil {
		t.Fatalf("CapturePage(...) = %v", err)
	}
	if got := len(q.Jobs()); got != 0 {
		t.Fatalf("rejected card queued %d details, want 0", got)
	}

	changed := dto.SearchConfig{UserID: target.UserID, UpdatedAt: first.UpdatedAt.Add(time.Second)}
	for range 2 {
		if err := svc.Reconsider(ctx, changed); err != nil {
			t.Fatalf("Reconsider(...) = %v", err)
		}
	}
	if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != card.URL {
		t.Fatalf("queued after reconsideration = %+v, want one job for %s", jobs, card.URL)
	}
}

var _ sourcetargets.QueuePublisher = (*failingJobQueue)(nil)
