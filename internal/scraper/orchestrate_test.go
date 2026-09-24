package scraper

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type pageSourceStub struct {
	stubSource
	next string
}

func (s *pageSourceStub) FetchPage(context.Context, string) ([]dto.Job, string, error) {
	return []dto.Job{{URL: s.urls[0]}}, s.next, nil
}

type emptyCandidateStore struct{}

func (emptyCandidateStore) SaveCards(context.Context, dto.SourceTarget, []dto.Job) ([]candidates.Candidate, error) {
	return nil, nil
}
func (emptyCandidateStore) ListForUser(context.Context, string, string, int) ([]candidates.Candidate, error) {
	return nil, nil
}
func (emptyCandidateStore) Assess(context.Context, string, string, time.Time, bool) (bool, error) {
	return false, nil
}
func (emptyCandidateStore) MarkDetailPending(context.Context, string) error { return nil }

func TestScrapePageStopsAtKnownJobFrontier(t *testing.T) {
	ctx := context.Background()
	url := "https://workinstartups.com/job/1"
	db := providers.NewMockJobProvider()
	if _, err := db.Save(ctx, []dto.Job{{URL: url}}); err != nil {
		t.Fatal(err)
	}
	src := &pageSourceStub{stubSource: stubSource{cfg: sources.Config{Name: "wis"}, urls: []string{url}}, next: "2:5"}
	q := queue.NewMockQueue()
	orch := New(nil, db, q).WithSourceReloader(nil, func(dto.SourceTarget) []sources.Source { return []sources.Source{src} }).WithCandidates(emptyCandidateStore{})
	next, err := orch.ScrapePage(ctx, dto.SourceTarget{ID: "target", UserID: "user", Source: "wis"}, "")
	if err != nil || next != "" {
		t.Fatalf("next=%q err=%v", next, err)
	}
}

// stubSource is an HTML-like source: implements Source and DetailFetcher so
// run() routes it through htmlPath (enqueue, not publish).
type stubSource struct {
	cfg   sources.Config
	urls  []string
	title string // optional; applied to every job's Title
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
		jobs[i] = dto.Job{URL: u, Title: s.title}
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

func TestRun_Enqueue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		urls     []string
		existing []string // URLs already saved in the DB
		want     []string // URLs expected in the queue
	}{
		{
			name: "enqueues new URLs",
			urls: []string{"https://example.com/job/1", "https://example.com/job/2"},
			want: []string{"https://example.com/job/1", "https://example.com/job/2"},
		},
		{
			name:     "filters URLs already in the DB",
			urls:     []string{"https://example.com/job/1", "https://example.com/job/2"},
			existing: []string{"https://example.com/job/1"},
			want:     []string{"https://example.com/job/2"},
		},
		{
			name: "deduplicates repeated URLs within a page",
			urls: []string{"https://example.com/job/1", "https://example.com/job/1"},
			want: []string{"https://example.com/job/1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := queue.NewMockQueue()
			db := providers.NewMockJobProvider()
			if len(tt.existing) > 0 {
				jobs := make([]dto.Job, len(tt.existing))
				for i, u := range tt.existing {
					jobs[i] = dto.Job{URL: u}
				}
				if _, err := db.Save(context.Background(), jobs); err != nil {
					t.Fatalf("seed existing jobs: %v", err)
				}
			}

			src := &stubSource{cfg: sources.Config{Name: "test"}, urls: tt.urls}

			o := New([]sources.Source{src}, db, q)
			if err := o.run(context.Background(), src); err != nil {
				t.Fatalf("run: %v", err)
			}

			if got := q.Items(); !slices.Equal(got, tt.want) {
				t.Errorf("queue = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunIfReady_IntervalGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		lastScrapedAgo time.Duration
		minInterval    time.Duration
		wantEnqueued   bool
	}{
		{
			name:           "skips when scraped within the interval",
			lastScrapedAgo: 0,
			minInterval:    time.Hour,
			wantEnqueued:   false,
		},
		{
			name:           "runs when the interval has elapsed",
			lastScrapedAgo: time.Hour,
			minInterval:    time.Millisecond,
			wantEnqueued:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := queue.NewMockQueue()
			db := providers.NewMockJobProvider()
			src := &stubSource{
				cfg:  sources.Config{Name: "test", MinScrapeInterval: tt.minInterval},
				urls: []string{"https://example.com/job/1"},
			}
			q.SetLastScrapedAt("test", time.Now().Add(-tt.lastScrapedAgo))

			o := New([]sources.Source{src}, db, q)
			o.runIfReady(context.Background(), src)

			if gotEnqueued := len(q.Items()) > 0; gotEnqueued != tt.wantEnqueued {
				t.Errorf("enqueued = %v, want %v", gotEnqueued, tt.wantEnqueued)
			}
		})
	}
}

func TestRunIfReady_SetsLastScraped(t *testing.T) {
	t.Parallel()

	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()
	src := &stubSource{
		cfg:  sources.Config{Name: "test", MinScrapeInterval: time.Hour},
		urls: []string{"https://example.com/job/1"},
	}

	o := New([]sources.Source{src}, db, q)
	o.runIfReady(context.Background(), src)

	if _, ok, _ := q.GetLastScraped(context.Background(), "test"); !ok {
		t.Error("expected last scraped to be recorded after run")
	}
}

func TestRun_MultiplePages(t *testing.T) {
	t.Parallel()

	q := queue.NewMockQueue()
	db := providers.NewMockJobProvider()

	page1 := []string{"https://example.com/job/1", "https://example.com/job/2"}
	page2 := []string{"https://example.com/job/3", "https://example.com/job/4"}

	src := &multiPageSource{pages: [][]string{page1, page2}}

	o := New([]sources.Source{src}, db, q)
	if err := o.run(context.Background(), src); err != nil {
		t.Fatalf("run: %v", err)
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
		name           string
		src            sources.Source
		wireFilter     bool
		excludeKeyword string // "" means the wired config has no exclusions
		wantExported   int
		wantQueued     int
	}{
		{
			name:         "ats publishes jobs",
			src:          atsSrc(2),
			wantExported: 2,
			wantQueued:   0,
		},
		{
			name:           "ats filter drops excluded title keyword",
			src:            atsSrc(2),
			wireFilter:     true,
			excludeKeyword: "engineer",
			wantExported:   0,
			wantQueued:     0,
		},
		{
			name:         "ats filter with no exclusions passes everything",
			src:          atsSrc(2),
			wireFilter:   true,
			wantExported: 2,
			wantQueued:   0,
		},
		{
			name:         "html enqueues not publishes",
			src:          htmlSrc(2),
			wantExported: 0,
			wantQueued:   2,
		},
		{
			name:           "html filter drops excluded title keyword",
			src:            &stubSource{cfg: sources.Config{Name: "html-test"}, urls: []string{"https://example.com/job/1"}, title: "Senior Engineer"},
			wireFilter:     true,
			excludeKeyword: "engineer",
			wantExported:   0,
			wantQueued:     0,
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
			if tt.wireFilter {
				cfg := dto.SearchConfig{UserID: "user1"}
				if tt.excludeKeyword != "" {
					cfg.ExcludedTitleKeywords = []string{tt.excludeKeyword}
				}
				o.WithRejectFilter(&stubCfgDB{cfgs: []dto.SearchConfig{cfg}})
			}

			if err := o.run(context.Background(), tt.src); err != nil {
				t.Fatalf("run: %v", err)
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

type stubCfgDB struct {
	cfgs []dto.SearchConfig
}

func (s *stubCfgDB) ListSearchConfigs(_ context.Context) ([]dto.SearchConfig, error) {
	return s.cfgs, nil
}
func (s *stubCfgDB) GetSearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	for _, cfg := range s.cfgs {
		if cfg.UserID == userID {
			return cfg, nil
		}
	}
	return dto.SearchConfig{}, nil
}
func (s *stubCfgDB) UpsertSearchConfig(_ context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	return cfg, nil
}

var _ providers.SearchConfigProvider = (*stubCfgDB)(nil)
