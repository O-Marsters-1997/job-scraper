package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type sourceQueueStub struct {
	mu     sync.Mutex
	items  []queue.SourceItem
	acked  int
	nacked int
}

func (q *sourceQueueStub) ClaimSource(context.Context, time.Duration) (queue.SourceItem, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return queue.SourceItem{}, false, nil
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true, nil
}

func (q *sourceQueueStub) AckSource(context.Context, queue.SourceItem) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.acked++
	return nil
}

func (q *sourceQueueStub) NackSource(context.Context, queue.SourceItem, string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.nacked++
	return nil
}
func (q *sourceQueueStub) RenewSource(context.Context, queue.SourceItem, time.Duration) error {
	return nil
}

func TestAcquisitionPoolRunsEligibleSourcesTogether(t *testing.T) {
	q := &sourceQueueStub{items: []queue.SourceItem{{ID: "one", Source: "wis"}, {ID: "two", Source: "remoteok"}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		RunAcquisition(ctx, q, 2, func(_ context.Context, item queue.SourceItem) error {
			started <- item.Source
			<-release
			return nil
		})
		close(done)
	}()
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("two sources did not run together")
		}
	}
	close(release)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pool did not stop")
	}
}

func TestCanceledAcquisitionIsRetried(t *testing.T) {
	q := &sourceQueueStub{items: []queue.SourceItem{{ID: "task", Source: "wis"}}}
	ctx, cancel := context.WithCancel(context.Background())
	RunAcquisition(ctx, q, 1, func(ctx context.Context, _ queue.SourceItem) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("handler context has no deadline")
		}
		cancel()
		return nil
	})
	if q.acked != 0 || q.nacked != 1 {
		t.Fatalf("canceled task: acked=%d nacked=%d", q.acked, q.nacked)
	}
}

type testDetailFetcher struct{}

func (testDetailFetcher) CanHandle(string) bool { return true }
func (testDetailFetcher) GetDetails(_ context.Context, url string) (dto.Job, error) {
	return dto.Job{Title: "Job", URL: url}, nil
}

func TestLegacyDetailStillDeliversToAPI(t *testing.T) {
	var delivered dto.Job
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ingest" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&delivered); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"status":"new"}`))
	}))
	defer server.Close()

	q := queue.NewMockQueue()
	url := "https://example.com/job"
	if err := q.EnqueueJobs(context.Background(), []dto.QueuedJob{{URL: url}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exporter := scraper.NewAPIExporter(server.URL, "token")
	if err := Run(ctx, q, func(ctx context.Context, job dto.QueuedJob) error {
		defer cancel()
		full, err := sources.Dispatch(ctx, []sources.DetailFetcher{testDetailFetcher{}}, job.URL)
		if err != nil {
			return err
		}
		return exporter.Export(ctx, full)
	}); err != nil {
		t.Fatal(err)
	}
	if delivered.URL != url || delivered.Title != "Job" {
		t.Fatalf("delivered = %+v", delivered)
	}
}

func TestSourceDetailDeliversToAPI(t *testing.T) {
	var delivered dto.Job
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ingest" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&delivered); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"status":"new"}`))
	}))
	defer server.Close()

	url := "https://example.com/job"
	payload, err := json.Marshal(dto.QueuedJob{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	q := &sourceQueueStub{items: []queue.SourceItem{{ID: "task", Source: "wis", Kind: queue.SourceDetail, Payload: payload}}}
	exporter := scraper.NewAPIExporter(server.URL, "token")
	runAcquisition(context.Background(), q, q.items[0], func(ctx context.Context, item queue.SourceItem) error {
		var job dto.QueuedJob
		if err := json.Unmarshal(item.Payload, &job); err != nil {
			return err
		}
		full, err := sources.Dispatch(ctx, []sources.DetailFetcher{testDetailFetcher{}}, job.URL)
		if err != nil {
			return err
		}
		return exporter.Export(ctx, full)
	})
	if delivered.URL != url || delivered.Title != "Job" {
		t.Fatalf("delivered = %+v", delivered)
	}
	if q.acked != 1 {
		t.Fatalf("source task acknowledged %d times", q.acked)
	}
}
