package discover

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

// fakeGate is an in-memory ScrapeGate double.
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
	companies []Company
	err       error
	calls     int
}

func (h *fakeHarvester) Name() string { return h.name }

func (h *fakeHarvester) Harvest(_ context.Context) ([]Company, error) {
	h.calls++
	if h.err != nil {
		return nil, h.err
	}
	return h.companies, nil
}

func TestRunner_UpsertsWithDerivedSlugs(t *testing.T) {
	h := &fakeHarvester{name: "yc", companies: []Company{
		{Name: "Acme Corp", Domain: "acme.com"},
		{Domain: "onlydomain.io"},
		{}, // no name, no domain: nothing to upsert
	}}
	companies := providers.NewMockCompanyProvider()
	r := NewRunner([]Harvester{h}, companies, newFakeGate())

	r.tick(context.Background())

	got, err := companies.ListCompaniesForUser(context.Background(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d companies, want 2: %+v", len(got), got)
	}
	bySlug := map[string]bool{}
	for _, c := range got {
		bySlug[c.Slug] = true
	}
	if !bySlug["acme-corp"] {
		t.Errorf("want slug acme-corp from Name, got %+v", got)
	}
	if !bySlug["onlydomainio"] {
		t.Errorf("want slug onlydomainio derived from Domain, got %+v", got)
	}
}

func TestRunner_HarvesterErrorSkipsOnlyThatHarvester(t *testing.T) {
	failing := &fakeHarvester{name: "broken", err: errors.New("boom")}
	ok := &fakeHarvester{name: "yc", companies: []Company{{Name: "Good Co"}}}
	companies := providers.NewMockCompanyProvider()
	gate := newFakeGate()
	r := NewRunner([]Harvester{failing, ok}, companies, gate)

	r.tick(context.Background())

	got, err := companies.ListCompaniesForUser(context.Background(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Good Co" {
		t.Fatalf("want only Good Co upserted, got %+v", got)
	}
	if _, ok, _ := gate.GetLastScraped(context.Background(), gateKeyPrefix+"broken"); ok {
		t.Error("failing harvester should not have its gate set")
	}
	if _, ok, _ := gate.GetLastScraped(context.Background(), gateKeyPrefix+"yc"); !ok {
		t.Error("succeeding harvester should have its gate set")
	}
}

func TestRunner_RespectsGate(t *testing.T) {
	h := &fakeHarvester{name: "yc", companies: []Company{{Name: "Acme"}}}
	companies := providers.NewMockCompanyProvider()
	gate := newFakeGate()
	gate.last[gateKeyPrefix+"yc"] = time.Now().Add(-time.Hour) // harvested 1h ago, well inside 24h gate

	r := NewRunner([]Harvester{h}, companies, gate)
	r.tick(context.Background())

	if h.calls != 0 {
		t.Errorf("want Harvest not called within gate window, got %d calls", h.calls)
	}
	got, _ := companies.ListCompaniesForUser(context.Background(), "")
	if len(got) != 0 {
		t.Errorf("want no upserts while gated, got %+v", got)
	}
}

func TestRunner_HarvestsWhenGateExpired(t *testing.T) {
	h := &fakeHarvester{name: "yc", companies: []Company{{Name: "Acme"}}}
	companies := providers.NewMockCompanyProvider()
	gate := newFakeGate()
	gate.last[gateKeyPrefix+"yc"] = time.Now().Add(-25 * time.Hour) // past the 24h gate

	r := NewRunner([]Harvester{h}, companies, gate)
	r.tick(context.Background())

	if h.calls != 1 {
		t.Errorf("want Harvest called once after gate expired, got %d calls", h.calls)
	}
}
