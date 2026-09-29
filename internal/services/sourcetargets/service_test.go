package sourcetargets_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

type failingPublisher struct {
	*queuetest.Recorder
	err error
}

func (f failingPublisher) Publish(context.Context, queue.Task) error { return f.err }

type fakeSearchConfigReader struct{}

func (fakeSearchConfigReader) SearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, data.ErrNotFound
}

type failingCandidateList struct {
	*jobsearchtest.FakeStore
	err error
}

func (f failingCandidateList) ListForUser(context.Context, string, string, int) ([]sourcetargets.Candidate, error) {
	return nil, f.err
}

func newService(targets sourcetargets.Store, q sourcetargets.QueuePublisher) *sourcetargets.Service {
	return sourcetargets.New(targets, fakeSearchConfigReader{}, q)
}

func TestCreate_RequiresSourceAndValue(t *testing.T) {
	svc := newService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("err = %v, want 400 apperr", err)
	}
}

func TestCreate_RejectsUnsupportedSource(t *testing.T) {
	svc := newService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "unknown-ats", Value: "x"})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("err = %v, want 400 apperr", err)
	}
}

func TestCreate_DiscoverySourceQueuesOneRun(t *testing.T) {
	q := queuetest.NewRecorder()
	svc := newService(jobsearchtest.NewFakeStore(), q)
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
	svc := newService(jobsearchtest.NewFakeStore(), failingPublisher{Recorder: queuetest.NewRecorder(), err: errors.New("queue unavailable")})
	target, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "wis", Value: "engineer"})
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if target.RunStatus != "queued" || target.RunID == "" {
		t.Fatalf("target = %+v, want a queued run", target)
	}
}

func TestCreate_PropagatesConflict(t *testing.T) {
	svc := newService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder())
	in := dto.CreateSourceTargetInput{Source: "wis", Value: "engineer", Enabled: boolPtr(false)}
	if _, err := svc.Create(context.Background(), "user-1", in); err != nil {
		t.Fatalf("first Create() err = %v", err)
	}
	if _, err := svc.Create(context.Background(), "user-1", in); wantStatus(t, err) != 409 {
		t.Fatalf("duplicate Create() err = %v, want 409", err)
	}
}

func TestCreate_IndeedURLsDifferingInTrackingParamsConflict(t *testing.T) {
	svc := newService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder())
	first := dto.CreateSourceTargetInput{Source: "indeed", Value: "https://www.indeed.com/jobs?q=golang&l=London&vjk=abc"}
	second := dto.CreateSourceTargetInput{Source: "indeed", Value: "https://www.indeed.com/jobs?l=London&q=golang&from=searchOnDesktopSerp&start=10"}
	target, err := svc.Create(context.Background(), "user-1", first)
	if err != nil {
		t.Fatalf("first Create() err = %v", err)
	}
	if want := "https://www.indeed.com/jobs?l=London&q=golang"; target.Value != want {
		t.Errorf("stored value = %q, want %q", target.Value, want)
	}
	if _, err := svc.Create(context.Background(), "user-1", second); wantStatus(t, err) != 409 {
		t.Fatalf("second Create() err = %v, want 409", err)
	}
}

func TestList_FillsBoardURL(t *testing.T) {
	svc := newService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder())
	in := dto.CreateSourceTargetInput{Source: "wis", Value: "engineer", Filters: map[string]string{"region": "uk"}, Enabled: boolPtr(false)}
	if _, err := svc.Create(context.Background(), "user-1", in); err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	got, err := svc.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List() err = %v", err)
	}
	if len(got) != 1 || got[0].URL != "https://workinstartups.com/search?q=engineer&w=uk" {
		t.Fatalf("List() = %+v, want one target with the board URL", got)
	}
}

func TestCreate_RejectsATSSource(t *testing.T) {
	svc := newService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder())
	_, err := svc.Create(context.Background(), "user-1", dto.CreateSourceTargetInput{Source: "greenhouse", Value: "acme"})
	if got := wantStatus(t, err); got != 400 {
		t.Fatalf("Create(greenhouse) status = %d, want 400", got)
	}
}

func createGreenhouseTarget(s *jobsearchtest.FakeStore) string {
	created, _ := s.CreateSourceTarget(context.Background(), "user-1", "greenhouse", "acme", true, nil)
	return created.ID
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		in         dto.UpdateSourceTargetInput
		targetID   func(*jobsearchtest.FakeStore) string
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
			targetID:   func(*jobsearchtest.FakeStore) string { return "missing-id" },
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := jobsearchtest.NewFakeStore()
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
	store := jobsearchtest.NewFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", false, nil)
	card := dto.Job{URL: "https://example.com/1", Title: "Engineer"}
	if _, err := store.SaveCards(context.Background(), created, []dto.Job{card}); err != nil {
		t.Fatalf("SaveCards() err = %v", err)
	}
	q := queuetest.NewRecorder()
	svc := newService(store, q)

	_, err := svc.Update(context.Background(), "user-1", dto.UpdateSourceTargetInput{ID: created.ID, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatalf("Update() err = %v", err)
	}
	if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != card.URL {
		t.Fatalf("queued after enabling = %+v, want one job for %s", jobs, card.URL)
	}
}

func TestUpdate_ReconsiderationFailureIsUnavailable(t *testing.T) {
	store := jobsearchtest.NewFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", false, nil)
	svc := sourcetargets.New(failingCandidateList{FakeStore: store, err: errors.New("boom")}, fakeSearchConfigReader{}, queuetest.NewRecorder())

	_, err := svc.Update(context.Background(), "user-1", dto.UpdateSourceTargetInput{ID: created.ID, Enabled: boolPtr(true)})
	if status, ok := apperr.StatusFor(err); !ok || status != 503 {
		t.Fatalf("err = %v, want 503 apperr", err)
	}
}

func TestScrape(t *testing.T) {
	store := jobsearchtest.NewFakeStore()
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
	store := jobsearchtest.NewFakeStore()
	created, _ := store.CreateSourceTarget(context.Background(), "user-1", "wis", "engineer", true, nil)
	started, _ := store.StartSourceTargetRun(context.Background(), created.ID)
	if _, err := store.TransitionSourceTargetRun(context.Background(), created.ID, started.RunID, "failed", "previous run failed"); err != nil {
		t.Fatalf("TransitionSourceTargetRun() err = %v", err)
	}
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
