package scraper

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// stubSource is an HTML-like source: implements Source and DetailFetcher so
// run() routes it through htmlPath (enqueue, not publish).
type stubSource struct {
	cfg  sources.Config
	urls []string
}

var _ sources.Source = (*stubSource)(nil)
var _ sources.DetailFetcher = (*stubSource)(nil)

func (s *stubSource) Cfg() sources.Config     { return s.cfg }
func (s *stubSource) CanHandle(_ string) bool { return false }
func (s *stubSource) GetDetails(_ context.Context, _ string) (dto.Job, error) {
	return dto.Job{}, nil
}
func (s *stubSource) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	jobs := make([]dto.Job, len(s.urls))
	for i, u := range s.urls {
		jobs[i] = dto.Job{URL: u}
	}
	_, err := fn(ctx, jobs)
	return err
}

// atsStubSource is an ATS-like source: implements Source only (not DetailFetcher)
// so run() routes it through atsPath (publish, not enqueue).
type atsStubSource struct {
	jobs []dto.Job
	name string
}

var _ sources.Source = (*atsStubSource)(nil)

func (s *atsStubSource) Cfg() sources.Config { return sources.Config{Name: s.name} }
func (s *atsStubSource) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	_, err := fn(ctx, s.jobs)
	return err
}

type recordingExporter struct {
	exported []dto.Job
}

func (r *recordingExporter) BulkExport(_ context.Context, jobs []dto.Job) error {
	r.exported = append(r.exported, jobs...)
	return nil
}

func TestRun_EnqueuesNewURLs(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	src := &stubSource{
		cfg:  sources.Config{Name: "test"},
		urls: []string{"https://example.com/job/1", "https://example.com/job/2"},
	}

	o := New([]sources.Source{src}, db, q)
	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	got := q.Items()
	want := []string{"https://example.com/job/1", "https://example.com/job/2"}
	if !slices.Equal(got, want) {
		t.Errorf("queue = %v, want %v", got, want)
	}
}

func TestRun_FiltersExistingURLs(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	_ = db.Save(context.Background(), []dto.Job{{URL: "https://example.com/job/1"}})

	src := &stubSource{
		cfg:  sources.Config{Name: "test"},
		urls: []string{"https://example.com/job/1", "https://example.com/job/2"},
	}

	o := New([]sources.Source{src}, db, q)
	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	got := q.Items()
	want := []string{"https://example.com/job/2"}
	if !slices.Equal(got, want) {
		t.Errorf("queue = %v, want %v", got, want)
	}
}

func TestRun_DeduplicatesWithinPage(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	src := &stubSource{
		cfg:  sources.Config{Name: "test"},
		urls: []string{"https://example.com/job/1", "https://example.com/job/1"},
	}

	o := New([]sources.Source{src}, db, q)
	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	got := q.Items()
	if len(got) != 1 {
		t.Errorf("queue has %d items, want 1 (dedup failed): %v", len(got), got)
	}
}

func TestRunIfReady_SkipsIfRecentlyScraped(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	src := &stubSource{
		cfg: sources.Config{
			Name:              "test",
			MinScrapeInterval: time.Hour,
		},
		urls: []string{"https://example.com/job/1"},
	}

	_ = q.SetLastScraped(context.Background(), "test")

	o := New([]sources.Source{src}, db, q)
	o.runIfReady(context.Background(), src)

	if len(q.Items()) != 0 {
		t.Errorf("expected no items enqueued after recent scrape, got %d", len(q.Items()))
	}
}

func TestRunIfReady_RunsIfNotRecentlyScraped(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	src := &stubSource{
		cfg: sources.Config{
			Name:              "test",
			MinScrapeInterval: time.Millisecond, // extremely short — already elapsed
		},
		urls: []string{"https://example.com/job/1"},
	}

	q.SetLastScrapedAt("test", time.Now().Add(-time.Hour))

	o := New([]sources.Source{src}, db, q)
	o.runIfReady(context.Background(), src)

	if len(q.Items()) == 0 {
		t.Error("expected URLs to be enqueued after interval elapsed")
	}
}

func TestRunIfReady_SetsLastScraped(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	src := &stubSource{
		cfg:  sources.Config{Name: "test", MinScrapeInterval: time.Hour},
		urls: []string{"https://example.com/job/1"},
	}

	o := New([]sources.Source{src}, db, q)
	o.runIfReady(context.Background(), src)

	_, ok, _ := q.GetLastScraped(context.Background(), "test")
	if !ok {
		t.Error("expected last scraped to be recorded after run")
	}
}

func TestRun_MultiplePages(t *testing.T) {
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()

	page1 := []string{"https://example.com/job/1", "https://example.com/job/2"}
	page2 := []string{"https://example.com/job/3", "https://example.com/job/4"}

	src := &multiPageSource{pages: [][]string{page1, page2}}

	o := New([]sources.Source{src}, db, q)
	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	got := q.Items()
	sort.Strings(got)
	want := []string{
		"https://example.com/job/1",
		"https://example.com/job/2",
		"https://example.com/job/3",
		"https://example.com/job/4",
	}
	if !slices.Equal(got, want) {
		t.Errorf("queue = %v, want %v", got, want)
	}
}

type multiPageSource struct {
	pages [][]string
}

var _ sources.Source = (*multiPageSource)(nil)
var _ sources.DetailFetcher = (*multiPageSource)(nil)

func (s *multiPageSource) Cfg() sources.Config {
	return sources.Config{Name: "multi"}
}
func (s *multiPageSource) CanHandle(_ string) bool { return false }
func (s *multiPageSource) GetDetails(_ context.Context, _ string) (dto.Job, error) {
	return dto.Job{}, nil
}
func (s *multiPageSource) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	for _, page := range s.pages {
		jobs := make([]dto.Job, len(page))
		for i, u := range page {
			jobs[i] = dto.Job{URL: u}
		}
		stop, err := fn(ctx, jobs)
		if err != nil || stop {
			return err
		}
	}
	return nil
}

func TestRun_Paths(t *testing.T) {
	t.Parallel()

	atsSrc := func(n int) *atsStubSource {
		jobs := make([]dto.Job, n)
		for i := range jobs {
			jobs[i] = dto.Job{
				Title: "Engineer",
				URL:   fmt.Sprintf("https://boards.greenhouse.io/acme/jobs/%d", i+1),
			}
		}
		return &atsStubSource{jobs: jobs, name: "ats-test"}
	}
	htmlSrc := func(n int) *stubSource {
		urls := make([]string, n)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://example.com/job/%d", i+1)
		}
		return &stubSource{cfg: sources.Config{Name: "html-test"}, urls: urls}
	}

	tests := []struct {
		name         string
		src          sources.Source
		useGate      bool
		wantExported int
		wantQueued   int
	}{
		{
			name:         "ats publishes jobs",
			src:          atsSrc(2),
			wantExported: 2,
			wantQueued:   0,
		},
		{
			name:         "ats gate drops below cutoff",
			src:          atsSrc(2),
			useGate:      true,
			wantExported: 0,
			wantQueued:   0,
		},
		{
			name:         "html enqueues not publishes",
			src:          htmlSrc(2),
			wantExported: 0,
			wantQueued:   2,
		},
		{
			name:         "html gate drops below cutoff",
			src:          htmlSrc(1),
			useGate:      true,
			wantExported: 0,
			wantQueued:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := queue.NewMockQueue()
			db := providers.NewMockJobProvider()
			exp := &recordingExporter{}

			o := New([]sources.Source{tt.src}, db, q)
			o.WithExporter(exp)
			if tt.useGate {
				// zeroScorer always scores 0; highCutoffCfgDB sets cutoff above 0,
				// so all jobs are dropped.
				o.WithRelevanceGate(&zeroScorer{}, &highCutoffCfgDB{}, nil, "user1")
			}

			if err := o.run(context.Background(), tt.src); err != nil {
				t.Fatal(err)
			}

			if got := len(exp.exported); got != tt.wantExported {
				t.Errorf("exported = %d, want %d", got, tt.wantExported)
			}
			if got := len(q.Items()); got != tt.wantQueued {
				t.Errorf("queued = %d, want %d", got, tt.wantQueued)
			}
		})
	}
}

// zeroScorer always returns 0.
type zeroScorer struct{}

func (z *zeroScorer) Score(_ dto.Job, _ dto.SearchConfig) int { return 0 }

// highCutoffCfgDB returns a SearchConfig with a high cutoff so zeroScorer always fails.
type highCutoffCfgDB struct{}

func (h *highCutoffCfgDB) GetSearchConfig(_ context.Context, _ string) (dto.SearchConfig, error) {
	return dto.SearchConfig{RelevanceCutoff: 100}, nil
}

func (h *highCutoffCfgDB) UpsertSearchConfig(_ context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	return cfg, nil
}

// Verify zeroScorer and highCutoffCfgDB satisfy the required interfaces.
var _ score.RelevanceScorer = (*zeroScorer)(nil)
var _ providers.SearchConfigProvider = (*highCutoffCfgDB)(nil)
