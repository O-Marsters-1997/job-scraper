package discover_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const gateKeyPrefix = "harvest:"

func runOnce(t *testing.T, gate *fakeGate, hs ...discover.Harvester) []dto.Company {
	t.Helper()
	store := jobsearchtest.NewFakeStore()
	_ = discover.NewRunner(hs, store, gate).RunOnce(t.Context())
	companies, err := store.ListCompaniesToCrawl(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	return companies
}

type fakeGate struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newFakeGate() *fakeGate { return &fakeGate{last: make(map[string]time.Time)} }

func (g *fakeGate) SetLastScraped(_ context.Context, source string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.last[source] = time.Now()
	return nil
}

func (g *fakeGate) GetLastScraped(_ context.Context, source string) (time.Time, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	t, ok := g.last[source]
	return t, ok, nil
}

type fakeHarvester struct {
	name      string
	companies []discover.Company
	err       error
	calls     int
}

func (h *fakeHarvester) Name() string { return h.name }

func (h *fakeHarvester) Harvest(_ context.Context) ([]discover.Company, error) {
	h.calls++
	if h.err != nil {
		return nil, h.err
	}
	return h.companies, nil
}

func TestRunner_UpsertsWithDerivedSlugs(t *testing.T) {
	h := &fakeHarvester{name: "yc", companies: []discover.Company{
		{Name: "Acme Corp", Domain: "acme.com"},
		{Domain: "onlydomain.io"},
		{},
	}}
	got := runOnce(t, newFakeGate(), h)

	var slugs []string
	for _, c := range got {
		slugs = append(slugs, c.Slug)
	}
	slices.Sort(slugs)
	if diff := cmp.Diff([]string{"acme-corp", "onlydomainio"}, slugs); diff != "" {
		t.Errorf("slugs (-want +got):\n%s", diff)
	}
}

func TestRunner_HarvesterErrorSkipsOnlyThatHarvester(t *testing.T) {
	failing := &fakeHarvester{name: "broken", err: errors.New("boom")}
	ok := &fakeHarvester{name: "yc", companies: []discover.Company{{Name: "Good Co"}}}
	gate := newFakeGate()

	got := runOnce(t, gate, failing, ok)

	if len(got) != 1 || got[0].Name != "Good Co" {
		t.Fatalf("want only Good Co upserted, got %+v", got)
	}
	if _, ok, _ := gate.GetLastScraped(t.Context(), gateKeyPrefix+"broken"); ok {
		t.Error("failing harvester should not have its gate set")
	}
	if _, ok, _ := gate.GetLastScraped(t.Context(), gateKeyPrefix+"yc"); !ok {
		t.Error("succeeding harvester should have its gate set")
	}
}

func TestRunner_Gate(t *testing.T) {
	for _, tt := range []struct {
		name          string
		lastRun       time.Duration
		wantHarvested int
	}{
		{"within the window is skipped", -time.Hour, 0},
		{"after the window harvests", -25 * time.Hour, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := &fakeHarvester{name: "yc", companies: []discover.Company{{Name: "Acme"}}}
			gate := newFakeGate()
			gate.last[gateKeyPrefix+"yc"] = time.Now().Add(tt.lastRun)

			got := runOnce(t, gate, h)

			if h.calls != tt.wantHarvested || len(got) != tt.wantHarvested {
				t.Errorf("harvest calls = %d, upserts = %d, want %d of each", h.calls, len(got), tt.wantHarvested)
			}
		})
	}
}

func TestRunner_RunOnceReturnsHarvesterErrors(t *testing.T) {
	boom := errors.New("boom")
	failing := &fakeHarvester{name: "broken", err: boom}
	err := discover.NewRunner([]discover.Harvester{failing}, jobsearchtest.NewFakeStore(), newFakeGate()).RunOnce(t.Context())
	if !errors.Is(err, boom) {
		t.Errorf("RunOnce() = %v, want it to wrap %v", err, boom)
	}
}
