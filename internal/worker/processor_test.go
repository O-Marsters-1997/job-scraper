package worker_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/worker"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type detailStub struct{ err error }

func (d detailStub) GetDetails(_ context.Context, url string) (dto.Job, error) {
	return dto.Job{Source: "wis", Title: "Job", URL: url}, d.err
}

type ingest struct {
	server   *httptest.Server
	requests atomic.Int32
}

func newIngest(t *testing.T, status int) *ingest {
	t.Helper()
	in := &ingest{}
	in.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		in.requests.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"status":"new"}`))
	}))
	t.Cleanup(in.server.Close)
	return in
}

type fixture struct {
	store     *jobsearchtest.FakeStore
	processor *worker.Processor
	ingest    *ingest
}

func newFixture(t *testing.T, ingestStatus int) fixture {
	t.Helper()
	store := jobsearchtest.NewFakeStore()
	js := jobsearch.Build(jobsearch.Deps{
		Store: store, SourceTargets: store,
		Scoring: jobsearchtest.NewNoopScoring(), Queue: jobsearchtest.NoopQueue{},
	})
	in := newIngest(t, ingestStatus)
	processor := worker.NewProcessor(worker.Deps{
		JS:       js,
		Exporter: scraper.NewAPIExporter(in.server.URL, "token").WithInitialBackoff(0),
		Detailers: map[string]sources.DetailFetcher{
			"wis":      detailStub{},
			"linkedin": detailStub{err: errors.New("page gone")},
		},
	})
	return fixture{store: store, processor: processor, ingest: in}
}

func detailTask(source string) queue.Task {
	return queue.Task{Version: 1, Source: source, Kind: queue.DetailTask, URL: "https://example.com/job", Card: dto.Job{Source: source}}
}

func (f fixture) startedRun(t *testing.T) dto.SourceTarget {
	t.Helper()
	ctx := context.Background()
	target, err := f.store.CreateSourceTarget(ctx, "user-1", "wis", "https://example.com/search", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	target, err = f.store.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestProcess(t *testing.T) {
	ctx := context.Background()

	t.Run("detail task exports the fetched job", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		if err := f.processor.Process(ctx, detailTask("wis")); err != nil {
			t.Fatal(err)
		}
		if got := f.ingest.requests.Load(); got != 1 {
			t.Fatalf("ingest requests = %d, want 1", got)
		}
	})

	t.Run("full feed card exports without a detail fetcher", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		if err := f.processor.Process(ctx, detailTask("remoteok")); err != nil {
			t.Fatal(err)
		}
		if got := f.ingest.requests.Load(); got != 1 {
			t.Fatalf("ingest requests = %d, want 1", got)
		}
	})

	t.Run("unsupported task kind fails without exporting", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		if err := f.processor.Process(ctx, queue.Task{Source: "wis", Kind: "bogus"}); err == nil {
			t.Fatal("poison task was acked")
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Fatalf("ingest requests = %d, want 0", got)
		}
	})

	t.Run("source without a detail fetcher fails without exporting", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		if err := f.processor.Process(ctx, detailTask("indeed")); err == nil {
			t.Fatal("poison task was acked")
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Fatalf("ingest requests = %d, want 0", got)
		}
	})

	t.Run("detail fetch failure is returned for requeue", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		if err := f.processor.Process(ctx, detailTask("linkedin")); err == nil {
			t.Fatal("failed fetch was acked")
		}
	})

	t.Run("transient ingest failure is returned for requeue", func(t *testing.T) {
		f := newFixture(t, http.StatusServiceUnavailable)
		if err := f.processor.Process(ctx, detailTask("wis")); err == nil {
			t.Fatal("failed export was acked")
		}
	})

	t.Run("redelivered page task for a finished run does not scrape again", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		target := f.startedRun(t)
		if _, err := f.store.TransitionSourceTargetRun(ctx, target.ID, target.RunID, "succeeded", ""); err != nil {
			t.Fatal(err)
		}
		task := queue.Task{Version: 1, Source: "wis", Kind: queue.ListingPageTask, TargetID: target.ID, RunID: target.RunID, Redelivered: true}
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		got, err := f.store.GetSourceTarget(ctx, target.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.RunStatus != "succeeded" || f.ingest.requests.Load() != 0 {
			t.Fatalf("run status = %s, ingest requests = %d", got.RunStatus, f.ingest.requests.Load())
		}
	})
}

func TestFailRun(t *testing.T) {
	ctx := context.Background()

	t.Run("marks the run failed", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		target := f.startedRun(t)
		task := queue.Task{Source: "wis", Kind: queue.ListingPageTask, TargetID: target.ID, RunID: target.RunID}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
		got, err := f.store.GetSourceTarget(ctx, target.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.RunStatus != "failed" || got.LastRunError == "" {
			t.Fatalf("run status = %q, error = %q", got.RunStatus, got.LastRunError)
		}
	})

	t.Run("detail task leaves runs alone", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		target := f.startedRun(t)
		task := queue.Task{Source: "wis", Kind: queue.DetailTask, TargetID: target.ID, RunID: target.RunID}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
		got, err := f.store.GetSourceTarget(ctx, target.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.RunStatus != "queued" {
			t.Fatalf("run status = %q, want queued", got.RunStatus)
		}
	})

	t.Run("run already gone is not an error", func(t *testing.T) {
		f := newFixture(t, http.StatusOK)
		task := queue.Task{Source: "wis", Kind: queue.ListingPageTask, TargetID: "gone", RunID: "gone"}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
	})
}
