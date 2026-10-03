package discover_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const gateKeyPrefix = "harvest:"

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

type recordingPublisher struct{ tasks []queue.Task }

func (p *recordingPublisher) Publish(_ context.Context, task queue.Task) error {
	p.tasks = append(p.tasks, task)
	return nil
}

type fakeHarvester struct {
	name     string
	interval time.Duration
	found    discover.Harvest
	err      error
	calls    int
}

func (h *fakeHarvester) Name() string            { return h.name }
func (h *fakeHarvester) Interval() time.Duration { return h.interval }

func (h *fakeHarvester) Harvest(context.Context) (discover.Harvest, error) {
	h.calls++
	return h.found, h.err
}

func runOnce(t *testing.T, gate *fakeGate, hs ...discover.Harvester) (*recordingPublisher, error) {
	t.Helper()
	return runOnceWith(t, jobsearchtest.NewFakeStore(), gate, hs...)
}

func runOnceWith(t *testing.T, catalog discover.Catalog, gate *fakeGate, hs ...discover.Harvester) (*recordingPublisher, error) {
	t.Helper()
	pub := &recordingPublisher{}
	err := discover.NewRunner(hs, pub, gate, catalog).RunOnce(t.Context())
	return pub, err
}

func TestRunner_PublishesBoardDiscoverTaskPerBoard(t *testing.T) {
	h := &fakeHarvester{name: "cc", interval: time.Hour, found: discover.Harvest{
		Boards: []discover.Board{{Source: "ashby", Token: "acme"}, {Source: "greenhouse", Token: "beta"}},
	}}

	pub, err := runOnce(t, newFakeGate(), h)
	if err != nil {
		t.Fatal(err)
	}

	var got []discover.Board
	for _, task := range pub.tasks {
		if task.Kind != queue.BoardDiscoverTask {
			t.Errorf("task kind = %q, want %q", task.Kind, queue.BoardDiscoverTask)
		}
		if err := task.Validate(); err != nil {
			t.Errorf("published task invalid: %v", err)
		}
		got = append(got, discover.Board{Source: task.Source, Token: task.BoardToken})
		if task.Via != h.name {
			t.Errorf("task via = %q, want %q", task.Via, h.name)
		}
	}
	if diff := cmp.Diff(h.found.Boards, got); diff != "" {
		t.Errorf("published boards (-want +got):\n%s", diff)
	}
}

func TestRunner_HarvesterErrorSkipsOnlyThatHarvester(t *testing.T) {
	boom := errors.New("boom")
	failing := &fakeHarvester{name: "broken", interval: time.Hour, err: boom}
	ok := &fakeHarvester{name: "cc", interval: time.Hour, found: discover.Harvest{Boards: []discover.Board{{Source: "ashby", Token: "good"}}}}
	gate := newFakeGate()

	pub, err := runOnce(t, gate, failing, ok)

	if !errors.Is(err, boom) {
		t.Errorf("RunOnce() = %v, want it to wrap %v", err, boom)
	}
	if len(pub.tasks) != 1 || pub.tasks[0].BoardToken != "good" {
		t.Fatalf("published = %+v, want only the good board", pub.tasks)
	}
	if _, ok, _ := gate.GetLastScraped(t.Context(), gateKeyPrefix+"broken"); ok {
		t.Error("failing harvester should not have its gate set")
	}
	if _, ok, _ := gate.GetLastScraped(t.Context(), gateKeyPrefix+"cc"); !ok {
		t.Error("succeeding harvester should have its gate set")
	}
}

func TestRunner_GateUsesEachHarvesterInterval(t *testing.T) {
	for _, tt := range []struct {
		name      string
		interval  time.Duration
		lastRun   time.Duration
		wantCalls int
	}{
		{"within its interval is skipped", 30 * 24 * time.Hour, -25 * time.Hour, 0},
		{"after its interval harvests", time.Hour, -2 * time.Hour, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := &fakeHarvester{name: "cc", interval: tt.interval}
			gate := newFakeGate()
			gate.last[gateKeyPrefix+"cc"] = time.Now().Add(tt.lastRun)

			if _, err := runOnce(t, gate, h); err != nil {
				t.Fatal(err)
			}

			if h.calls != tt.wantCalls {
				t.Errorf("harvest calls = %d, want %d", h.calls, tt.wantCalls)
			}
		})
	}
}

func wttjCompany(token string) discover.Company {
	return discover.Company{Slug: token, Name: "Derived", Board: discover.Board{Source: "wttj", Token: token}}
}

func TestRunner_RepublishesDiscoveryForCandidateBoards(t *testing.T) {
	catalog := jobsearchtest.NewFakeStore()
	h := &fakeHarvester{name: "wttj", interval: time.Hour, found: discover.Harvest{Companies: []discover.Company{wttjCompany("faculty")}}}

	var published []queue.Task
	for range 2 {
		gate := newFakeGate()
		pub, err := runOnceWith(t, catalog, gate, h)
		if err != nil {
			t.Fatal(err)
		}
		if len(pub.tasks) != 1 || pub.tasks[0].Kind != queue.BoardDiscoverTask || pub.tasks[0].BoardToken != "faculty" {
			t.Fatalf("published = %+v, want one board_discover for faculty", pub.tasks)
		}
		if err := pub.tasks[0].Validate(); err != nil {
			t.Errorf("published task invalid: %v", err)
		}
		published = append(published, pub.tasks[0])
	}

	company, err := catalog.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "faculty", Name: "Derived"})
	if err != nil {
		t.Fatal(err)
	}
	boards, _ := catalog.ListCompanyBoards(t.Context(), company.ID)
	if len(boards) != 1 || boards[0].Status != dto.BoardCandidate {
		t.Errorf("boards after two runs = %+v, want one candidate", boards)
	}
	if got := published[0].CompanyID; got != company.ID {
		t.Errorf("task CompanyID = %q, want %q", got, company.ID)
	}
}

func TestRunner_VerifiedBoardIsNotQueuedAgain(t *testing.T) {
	catalog := jobsearchtest.NewFakeStore()
	company, _ := catalog.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "faculty", Name: "Faculty"})
	_, _ = catalog.UpsertCandidateBoard(t.Context(), company.ID, "wttj", "faculty")
	_, _ = catalog.VerifyCompanyBoard(t.Context(), company.ID, "wttj", "faculty", "wttj-origin", "")
	h := &fakeHarvester{name: "wttj", interval: time.Hour, found: discover.Harvest{Companies: []discover.Company{wttjCompany("faculty")}}}

	pub, err := runOnceWith(t, catalog, newFakeGate(), h)
	if err != nil {
		t.Fatal(err)
	}

	if len(pub.tasks) != 0 {
		t.Errorf("published = %+v, want none", pub.tasks)
	}
}

func TestRunner_CompanyWithVerifiedOtherBoardGetsNoCandidate(t *testing.T) {
	catalog := jobsearchtest.NewFakeStore()
	company, _ := catalog.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "faculty", Name: "Faculty"})
	_, _ = catalog.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "faculty")
	_, _ = catalog.VerifyCompanyBoard(t.Context(), company.ID, "greenhouse", "faculty", "discovered", "")
	h := &fakeHarvester{name: "wttj", interval: time.Hour, found: discover.Harvest{Companies: []discover.Company{wttjCompany("faculty")}}}

	pub, err := runOnceWith(t, catalog, newFakeGate(), h)
	if err != nil {
		t.Fatal(err)
	}

	boards, _ := catalog.ListCompanyBoards(t.Context(), company.ID)
	if len(boards) != 1 || boards[0].Source != "greenhouse" {
		t.Errorf("boards = %+v, want only the greenhouse board", boards)
	}
	if len(pub.tasks) != 0 {
		t.Errorf("published = %+v, want none", pub.tasks)
	}
}

func TestRunner_VerifiedHarvestedBoardIsSkippedUnlessRecheck(t *testing.T) {
	for _, tt := range []struct {
		name    string
		recheck bool
		want    int
	}{
		{"verified board is dropped", false, 0},
		{"recheck republishes it", true, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			catalog := jobsearchtest.NewFakeStore()
			company, _ := catalog.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
			_, _ = catalog.UpsertCandidateBoard(t.Context(), company.ID, "ashby", "acme")
			_, _ = catalog.VerifyCompanyBoard(t.Context(), company.ID, "ashby", "acme", "ashby-origin", "")
			h := &fakeHarvester{name: "x", interval: time.Hour, found: discover.Harvest{
				Boards:  []discover.Board{{Source: "ashby", Token: "acme"}},
				Recheck: tt.recheck,
			}}

			pub, err := runOnceWith(t, catalog, newFakeGate(), h)
			if err != nil {
				t.Fatal(err)
			}

			if len(pub.tasks) != tt.want {
				t.Errorf("published = %+v, want %d", pub.tasks, tt.want)
			}
		})
	}
}
