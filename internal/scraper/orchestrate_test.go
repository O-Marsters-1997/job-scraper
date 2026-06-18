package scraper

import (
	"context"
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

type recordingPublisher struct {
	published []dto.Job
}

func (r *recordingPublisher) Publish(_ context.Context, jobs []dto.Job) error {
	r.published = append(r.published, jobs...)
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

func TestATSPath_PublishesJobs(t *testing.T) {
	pub := &recordingPublisher{}
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()

	jobs := []dto.Job{
		{Title: "Engineer", URL: "https://boards.greenhouse.io/acme/jobs/1"},
		{Title: "Designer", URL: "https://boards.greenhouse.io/acme/jobs/2"},
	}
	src := &atsStubSource{jobs: jobs, name: "ats-test"}

	o := New([]sources.Source{src}, db, q)
	o.WithPublisher(pub)

	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	if len(pub.published) != 2 {
		t.Errorf("published %d jobs, want 2", len(pub.published))
	}
	if len(q.Items()) != 0 {
		t.Errorf("expected nothing enqueued for ATS path, got %d", len(q.Items()))
	}
}

func TestATSPath_RelevanceGateDropsBelowCutoff(t *testing.T) {
	pub := &recordingPublisher{}
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()

	jobs := []dto.Job{
		{Title: "Engineer", URL: "https://boards.greenhouse.io/acme/jobs/1"},
		{Title: "Designer", URL: "https://boards.greenhouse.io/acme/jobs/2"},
	}
	src := &atsStubSource{jobs: jobs, name: "ats-gate-test"}

	o := New([]sources.Source{src}, db, q)
	o.WithPublisher(pub)
	// zeroScorer always scores 0; cutoff in mock config will be above 0 so all jobs are dropped.
	o.WithRelevanceGate(&zeroScorer{}, &highCutoffCfgDB{}, nil, "user1")

	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	if len(pub.published) != 0 {
		t.Errorf("expected 0 published (all below cutoff), got %d", len(pub.published))
	}
}

func TestHTMLPath_EnqueuesNotPublishes(t *testing.T) {
	pub := &recordingPublisher{}
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()

	src := &stubSource{
		cfg:  sources.Config{Name: "html-test"},
		urls: []string{"https://example.com/job/1", "https://example.com/job/2"},
	}

	o := New([]sources.Source{src}, db, q)
	o.WithPublisher(pub)

	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	if len(q.Items()) != 2 {
		t.Errorf("expected 2 items enqueued, got %d", len(q.Items()))
	}
	if len(pub.published) != 0 {
		t.Errorf("expected nothing published directly for HTML path, got %d", len(pub.published))
	}
}

func TestHTMLPath_RelevanceGateDropsBelowCutoff(t *testing.T) {
	pub := &recordingPublisher{}
	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()

	src := &stubSource{
		cfg:  sources.Config{Name: "html-gate-test"},
		urls: []string{"https://example.com/job/1"},
	}

	o := New([]sources.Source{src}, db, q)
	o.WithPublisher(pub)
	o.WithRelevanceGate(&zeroScorer{}, &highCutoffCfgDB{}, nil, "user1")

	if err := o.run(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	if len(q.Items()) != 0 {
		t.Errorf("expected 0 items enqueued (all below cutoff), got %d", len(q.Items()))
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
