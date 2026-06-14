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
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type stubSource struct {
	cfg  sources.Config
	urls []string
}

func (s *stubSource) Cfg() sources.Config     { return s.cfg }
func (s *stubSource) CanHandle(_ string) bool { return false }
func (s *stubSource) NeedsDetail() bool       { return true }
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

func (s *multiPageSource) Cfg() sources.Config {
	return sources.Config{Name: "multi"}
}
func (s *multiPageSource) CanHandle(_ string) bool { return false }
func (s *multiPageSource) NeedsDetail() bool       { return true }
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
