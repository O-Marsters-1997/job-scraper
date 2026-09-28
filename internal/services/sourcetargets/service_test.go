package sourcetargets_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

type failingPublisher struct{ err error }

func (f failingPublisher) Publish(context.Context, queue.Task) error { return f.err }

var errSourceTargetExists = apperr.Conflict("source target already exists")

type fakeSearchConfigReader struct{}

func (fakeSearchConfigReader) SearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, data.ErrNotFound
}

type fakeStore struct {
	mu        sync.Mutex
	targets   map[string]dto.SourceTarget
	CreateErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{targets: make(map[string]dto.SourceTarget)}
}

func (f *fakeStore) GetVerifiedBoardID(context.Context, string, string) (string, error) {
	return "", apperr.NotFound("board not verified")
}

func (f *fakeStore) create(userID, source, value string, enabled bool, withRun bool) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.CreateErr != nil {
		return dto.SourceTarget{}, f.CreateErr
	}
	for _, t := range f.targets {
		if t.UserID == userID && t.Source == source && t.Value == value {
			return dto.SourceTarget{}, errSourceTargetExists
		}
	}
	target := dto.SourceTarget{
		ID: fmt.Sprintf("target-%d", len(f.targets)+1), UserID: userID, Source: source, Value: value, Enabled: enabled,
	}
	if withRun {
		target.RunID = "run-1"
		target.RunStatus = "queued"
	}
	f.targets[target.ID] = target
	return target, nil
}

func (f *fakeStore) CreateSourceTarget(_ context.Context, userID, source, value string, enabled bool, _ map[string]string) (dto.SourceTarget, error) {
	return f.create(userID, source, value, enabled, false)
}

func (f *fakeStore) CreateSourceTargetWithRun(_ context.Context, userID, source, value string, enabled bool, _ map[string]string) (dto.SourceTarget, error) {
	return f.create(userID, source, value, enabled, true)
}

func (f *fakeStore) UpdateSourceTarget(_ context.Context, id, userID string, enabled *bool, checkIntervalMinutes *int) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.targets[id]
	if !ok || target.UserID != userID {
		return dto.SourceTarget{}, apperr.NotFound("not found")
	}
	if enabled != nil {
		target.Enabled = *enabled
	}
	if checkIntervalMinutes != nil {
		target.CheckIntervalMinutes = *checkIntervalMinutes
	}
	f.targets[id] = target
	return target, nil
}

func (f *fakeStore) DeleteSourceTarget(_ context.Context, id, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.targets, id)
	return nil
}

func (f *fakeStore) ListSourceTargetsByUser(_ context.Context, userID string) ([]dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.SourceTarget
	for _, t := range f.targets {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) StartSourceTargetRun(_ context.Context, id string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.targets[id]
	if !ok {
		return dto.SourceTarget{}, apperr.NotFound("not found")
	}
	target.RunID = "run-" + id
	target.RunStatus = "queued"
	target.LastRunError = ""
	target.Enabled = true
	f.targets[id] = target
	return target, nil
}

func (f *fakeStore) GetSourceTarget(_ context.Context, id string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.targets[id]
	if !ok {
		return dto.SourceTarget{}, apperr.NotFound("not found")
	}
	return target, nil
}

func (f *fakeStore) TransitionSourceTargetRun(_ context.Context, id, runID, status, runError string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.targets[id]
	if !ok || target.RunID != runID {
		return dto.SourceTarget{}, apperr.NotFound("not found")
	}
	target.RunStatus, target.LastRunError = status, runError
	f.targets[id] = target
	return target, nil
}

func (f *fakeStore) ListRecoverableSourceTargets(context.Context) ([]dto.SourceTarget, error) {
	return nil, nil
}

func (f *fakeStore) ClaimRecoverableSourceTarget(_ context.Context, id, runID string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.targets[id]
	if !ok || target.RunID != runID {
		return dto.SourceTarget{}, apperr.NotFound("not found")
	}
	return target, nil
}

func (f *fakeStore) forceRunState(id, status, runError string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target := f.targets[id]
	target.RunStatus = status
	target.LastRunError = runError
	f.targets[id] = target
}

type fakeReconsiderer struct{ err error }

func (f fakeReconsiderer) Reconsider(context.Context, dto.SearchConfig) error { return f.err }

func newService(targets *fakeStore, q sourcetargets.QueuePublisher) *sourcetargets.Service {
	return sourcetargets.New(targets, fakeSearchConfigReader{}, fakeReconsiderer{}, q)
}

func TestCreate_RequiresSourceAndValue(t *testing.T) {
	svc := newService(newFakeStore(), queuetest.NewRecorder())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("err = %v, want 400 apperr", err)
	}
}

func TestCreate_RejectsUnsupportedSource(t *testing.T) {
	svc := newService(newFakeStore(), queuetest.NewRecorder())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "unknown-ats", Value: "x"})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("err = %v, want 400 apperr", err)
	}
}

func TestCreate_DiscoverySourceQueuesOneRun(t *testing.T) {
	q := queuetest.NewRecorder()
	svc := newService(newFakeStore(), q)
	target, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "wis", Value: "engineer"})
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if target.RunStatus != "queued" {
		t.Fatalf("run status = %q, want queued", target.RunStatus)
	}
	if got := q.Tasks(); len(got) != 1 || got[0].Source != "wis" {
		t.Fatalf("queued tasks = %+v, want one WIS search", got)
	}
}

func TestCreate_KeepsRecoverableRunAfterQueueFailure(t *testing.T) {
	svc := newService(newFakeStore(), failingPublisher{err: errors.New("queue unavailable")})
	target, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "wis", Value: "engineer"})
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if target.RunStatus != "queued" || target.RunID == "" {
		t.Fatalf("target = %+v, want a queued run", target)
	}
}

func TestCreate_PropagatesConflict(t *testing.T) {
	store := newFakeStore()
	store.CreateErr = errSourceTargetExists
	svc := newService(store, queuetest.NewRecorder())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "greenhouse", Value: "acme"})
	if !errors.Is(err, errSourceTargetExists) {
		t.Fatalf("err = %v, want ErrSourceTargetExists", err)
	}
}

func createGreenhouseTarget(s *fakeStore) string {
	created, _ := s.CreateSourceTarget(context.Background(), "user-1", "greenhouse", "acme", true, nil)
	return created.ID
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		in         dto.UpdateSourceTargetInput
		targetID   func(*fakeStore) string
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
			targetID:   func(*fakeStore) string { return "missing-id" },
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			id := tt.targetID(store)
			svc := newService(store, queuetest.NewRecorder())

			tt.in.ID = id
			_, err := svc.Update(context.Background(), "user-1", tt.in)
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
	store := newFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", false, nil)
	svc := newService(store, queuetest.NewRecorder())

	_, err := svc.Update(context.Background(), "user-1", dto.UpdateSourceTargetInput{ID: created.ID, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatalf("Update() err = %v", err)
	}
}

func TestUpdate_ReconsiderationFailureIsUnavailable(t *testing.T) {
	store := newFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", false, nil)
	svc := sourcetargets.New(store, fakeSearchConfigReader{}, fakeReconsiderer{err: errors.New("boom")}, queuetest.NewRecorder())

	_, err := svc.Update(context.Background(), "user-1", dto.UpdateSourceTargetInput{ID: created.ID, Enabled: boolPtr(true)})
	if status, ok := apperr.StatusFor(err); !ok || status != 503 {
		t.Fatalf("err = %v, want 503 apperr", err)
	}
}

func TestScrape(t *testing.T) {
	store := newFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", true, nil)
	svc := newService(store, queuetest.NewRecorder())

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
	store := newFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", true, nil)
	store.forceRunState(created.ID, "failed", "previous run failed")
	q := queuetest.NewRecorder()
	svc := newService(store, q)

	target, err := svc.Scrape(context.Background(), "user-1", created.ID)
	if err != nil {
		t.Fatalf("Scrape() err = %v", err)
	}
	if target.RunStatus != "queued" || target.LastRunError != "" {
		t.Fatalf("run state after retry = %+v", target)
	}
	if len(q.Tasks()) != 1 {
		t.Fatalf("queued tasks = %d, want 1", len(q.Tasks()))
	}
}
