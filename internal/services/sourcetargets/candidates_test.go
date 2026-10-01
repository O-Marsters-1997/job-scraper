package sourcetargets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
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

	t.Run("drops LinkedIn cards for tracked companies before detail fetch", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		q := queuetest.NewRecorder()
		svc := sourcetargets.New(st, fakeSearchConfigReader{}, q, verifiedBoards{{CompanySlug: "polled-co", Source: "ashby", BoardToken: "polled", Tracked: true}})
		linkedin := dto.SourceTarget{ID: "target-2", UserID: userID, Source: "linkedin"}
		polled := dto.Job{URL: "https://linkedin.com/jobs/view/1", Title: "Engineer", CompanySlug: "polled-co"}
		unpolled := dto.Job{URL: "https://linkedin.com/jobs/view/2", Title: "Engineer", CompanySlug: "other-co"}
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), linkedin, []dto.Job{polled, unpolled}, config); err != nil {
			t.Fatalf("CapturePage(linkedin) err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != unpolled.URL {
			t.Errorf("queued after linkedin page = %+v, want only %s", jobs, unpolled.URL)
		}

		other := dto.Job{URL: "https://example.com/3", Title: "Engineer", CompanySlug: "polled-co"}
		if err := svc.CapturePage(t.Context(), captureTarget, []dto.Job{other}, config); err != nil {
			t.Fatalf("CapturePage(wis) err = %v", err)
		}
		if got := len(q.Jobs()); got != 2 {
			t.Errorf("queued %d details after non-linkedin card, want 2", got)
		}
		if got := q.Tasks(); len(got) != 0 {
			t.Errorf("published %+v for tracked company, want nothing", got)
		}
	})

	t.Run("drops LinkedIn cards for untracked verified companies and harvests the board once", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		q := queuetest.NewRecorder()
		svc := sourcetargets.New(st, fakeSearchConfigReader{}, q, verifiedBoards{{CompanySlug: "acme", Source: "ashby", BoardToken: "acme"}})
		linkedin := dto.SourceTarget{ID: "target-2", UserID: userID, Source: "linkedin"}
		cards := []dto.Job{
			{URL: "https://linkedin.com/jobs/view/1", Title: "Engineer", CompanySlug: "acme"},
			{URL: "https://linkedin.com/jobs/view/2", Title: "Designer", CompanySlug: "acme"},
			{URL: "https://linkedin.com/jobs/view/3", Title: "Engineer", CompanySlug: "other-co"},
		}
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), linkedin, cards, config); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != cards[2].URL {
			t.Errorf("queued = %+v, want only %s", jobs, cards[2].URL)
		}
		tasks := q.Tasks()
		if len(tasks) != 1 || tasks[0].Kind != queue.BoardDiscoverTask || tasks[0].Source != "ashby" || tasks[0].BoardToken != "acme" {
			t.Errorf("published = %+v, want one board_discover for ashby/acme", tasks)
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
