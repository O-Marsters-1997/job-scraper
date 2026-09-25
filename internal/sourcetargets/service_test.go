package sourcetargets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sourcetargets"
)

// emptyStore is a candidates.Store with nothing in it, so Reconsider is a no-op.
type emptyStore struct{}

func (emptyStore) SaveCards(context.Context, dto.SourceTarget, []dto.Job) ([]candidates.Candidate, error) {
	return nil, nil
}
func (emptyStore) ListForUser(context.Context, string, string, int) ([]candidates.Candidate, error) {
	return nil, nil
}
func (emptyStore) Assess(context.Context, string, string, time.Time, bool) (bool, error) {
	return true, nil
}
func (emptyStore) MarkDetailPending(context.Context, string) error { return nil }

func newService(targets providers.SourceTargetProvider, q *queue.MockQueue) *sourcetargets.Service {
	candidateSvc := candidates.New(emptyStore{}, nil)
	configs := providers.NewMockSearchConfigProvider()
	return sourcetargets.New(targets, configs, candidateSvc, q)
}

func TestCreate_RequiresSourceAndValue(t *testing.T) {
	svc := newService(providers.NewMockSourceTargetProvider(), queue.NewMockQueue())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("err = %v, want 400 apperr", err)
	}
}

func TestCreate_RejectsUnsupportedSource(t *testing.T) {
	svc := newService(providers.NewMockSourceTargetProvider(), queue.NewMockQueue())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "unknown-ats", Value: "x"})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("err = %v, want 400 apperr", err)
	}
}

func TestCreate_DiscoverySourceQueuesOneRun(t *testing.T) {
	q := queue.NewMockQueue()
	svc := newService(providers.NewMockSourceTargetProvider(), q)
	target, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "wis", Value: "engineer"})
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if target.RunStatus != "queued" {
		t.Fatalf("run status = %q, want queued", target.RunStatus)
	}
	if got := q.ScrapeRequests(); len(got) != 1 || got[0].Target.Source != "wis" {
		t.Fatalf("queued requests = %+v, want one WIS search", got)
	}
}

func TestCreate_KeepsRecoverableRunAfterQueueFailure(t *testing.T) {
	q := queue.NewMockQueue()
	q.EnqueueScrapeErr = errors.New("queue unavailable")
	svc := newService(providers.NewMockSourceTargetProvider(), q)
	target, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "wis", Value: "engineer"})
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if target.RunStatus != "queued" || target.RunID == "" {
		t.Fatalf("target = %+v, want a queued run", target)
	}
	if len(q.ScrapeRequests()) != 0 {
		t.Fatalf("queued requests = %+v, want none", q.ScrapeRequests())
	}
}

func TestCreate_PropagatesConflict(t *testing.T) {
	store := providers.NewMockSourceTargetProvider()
	store.CreateErr = providers.ErrSourceTargetExists
	svc := newService(store, queue.NewMockQueue())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "greenhouse", Value: "acme"})
	if !errors.Is(err, providers.ErrSourceTargetExists) {
		t.Fatalf("err = %v, want ErrSourceTargetExists", err)
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		in         dto.UpdateSourceTargetInput
		targetID   func(*providers.MockSourceTargetProvider) string
		wantStatus int
	}{
		{
			name:     "updates enabled flag",
			in:       dto.UpdateSourceTargetInput{Enabled: boolPtr(false)},
			targetID: createGreenhouseTarget,
		},
		{
			name:     "updates check interval",
			in:       dto.UpdateSourceTargetInput{CheckIntervalMinutes: intPtr(60)},
			targetID: createGreenhouseTarget,
		},
		{
			name:       "rejects interval below 60",
			in:         dto.UpdateSourceTargetInput{CheckIntervalMinutes: intPtr(30)},
			targetID:   createGreenhouseTarget,
			wantStatus: 400,
		},
		{
			name:       "rejects empty body",
			in:         dto.UpdateSourceTargetInput{},
			targetID:   createGreenhouseTarget,
			wantStatus: 400,
		},
		{
			name:       "not found",
			in:         dto.UpdateSourceTargetInput{Enabled: boolPtr(false)},
			targetID:   func(*providers.MockSourceTargetProvider) string { return "missing-id" },
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSourceTargetProvider()
			id := tt.targetID(store)
			svc := newService(store, queue.NewMockQueue())

			_, err := svc.Update(context.Background(), "user-1", id, tt.in)
			if tt.wantStatus == 0 {
				if err != nil {
					t.Fatalf("Update() err = %v", err)
				}
				return
			}
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("err = %v, want status %d", err, tt.wantStatus)
			}
		})
	}
}

func TestUpdate_EnablingDiscoveryTargetReconsidersCandidates(t *testing.T) {
	store := providers.NewMockSourceTargetProvider()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", false, nil)
	svc := newService(store, queue.NewMockQueue())

	_, err := svc.Update(context.Background(), "user-1", created.ID, dto.UpdateSourceTargetInput{Enabled: boolPtr(true)})
	if err != nil {
		t.Fatalf("Update() err = %v", err)
	}
}

func createGreenhouseTarget(s *providers.MockSourceTargetProvider) string {
	created, _ := s.CreateSourceTarget(context.Background(), "user-1", "greenhouse", "acme", true, nil)
	return created.ID
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestScrape(t *testing.T) {
	store := providers.NewMockSourceTargetProvider()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", true, nil)
	svc := newService(store, queue.NewMockQueue())

	if _, err := svc.Scrape(context.Background(), "user-2", created.ID); wantStatus(t, err) != 404 {
		t.Fatalf("cross-user scrape err = %v, want 404", err)
	}

	target, err := svc.Scrape(context.Background(), "user-1", created.ID)
	if err != nil {
		t.Fatalf("Scrape() err = %v", err)
	}
	if target.RunStatus != "queued" {
		t.Fatalf("run status = %q, want queued", target.RunStatus)
	}

	if _, err := svc.Scrape(context.Background(), "user-1", created.ID); wantStatus(t, err) != 409 {
		t.Fatalf("rerun while queued err = %v, want 409", err)
	}
}

func wantStatus(t *testing.T, err error) int {
	t.Helper()
	status, ok := apperr.StatusFor(err)
	if !ok {
		t.Fatalf("err = %v, has no apperr kind", err)
	}
	return status
}

func TestScrape_RetriesFailedRun(t *testing.T) {
	store := providers.NewMockSourceTargetProvider()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", true, nil)
	_, _ = store.SetSourceTargetRunState(context.Background(), created.ID, "failed", "previous run failed")
	q := queue.NewMockQueue()
	svc := newService(store, q)

	target, err := svc.Scrape(context.Background(), "user-1", created.ID)
	if err != nil {
		t.Fatalf("Scrape() err = %v", err)
	}
	if target.RunStatus != "queued" || target.LastRunError != "" {
		t.Fatalf("run state after retry = %+v", target)
	}
	if len(q.ScrapeRequests()) != 1 {
		t.Fatalf("queued requests = %d, want 1", len(q.ScrapeRequests()))
	}
}
