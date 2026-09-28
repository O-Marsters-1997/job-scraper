package scoringconfig_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

type fakeStore struct {
	mu      sync.Mutex
	configs map[string]dto.SearchConfig
	options []dto.ScoringOption

	GetErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{configs: make(map[string]dto.SearchConfig)}
}

func (f *fakeStore) seedOptions() *fakeStore {
	f.options = []dto.ScoringOption{
		{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
		{ID: "tech:kubernetes", Dimension: dto.DimensionTech, Label: "Kubernetes", Question: "Does the role use Kubernetes?"},
		{ID: "domain:gambling", Dimension: dto.DimensionDomain, Label: "Gambling", Question: "Is the company's main business gambling?"},
		{ID: "seniority:senior", Dimension: dto.DimensionSeniority, Label: "Senior", Question: "Seniority?"},
	}
	return f
}

func (f *fakeStore) GetSearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	if f.GetErr != nil {
		return dto.SearchConfig{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg, ok := f.configs[userID]
	if !ok {
		return dto.SearchConfig{}, apperr.NotFound("not found")
	}
	return cfg, nil
}

func (f *fakeStore) UpsertSearchConfig(_ context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.configs[cfg.UserID] = cfg
	return cfg, nil
}

func (f *fakeStore) ListScoringOptions(context.Context) ([]dto.ScoringOption, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.ScoringOption, len(f.options))
	copy(out, f.options)
	return out, nil
}

type fakeReconsiderer struct {
	err        error
	calledWith dto.SearchConfig
}

func (f *fakeReconsiderer) Reconsider(_ context.Context, config dto.SearchConfig) error {
	f.calledWith = config
	return f.err
}

type fakeRecomputer struct {
	err        error
	calledWith string
	calls      int
}

func (f *fakeRecomputer) Recompute(_ context.Context, userID string) (dto.RecomputeResult, error) {
	f.calledWith = userID
	f.calls++
	return dto.RecomputeResult{}, f.err
}

type fakeExtractor struct {
	calls      int
	gotText    string
	gotOptions []dto.ScoringOption
	picks      []dto.Pick
	err        error
}

func (f *fakeExtractor) Extract(_ context.Context, _ string, text string, options []dto.ScoringOption, _ []dto.DimensionSpec) ([]dto.Pick, error) {
	f.calls++
	f.gotText = text
	f.gotOptions = options
	return f.picks, f.err
}

type fakeBackfiller struct {
	queued     int64
	err        error
	calledWith string
	calls      int
}

func (f *fakeBackfiller) FillMissingAnswers(_ context.Context, userID string) (int64, error) {
	f.calledWith = userID
	f.calls++
	return f.queued, f.err
}

type fakeCredentials struct {
	key string
	err error
}

func (f *fakeCredentials) Get(_ context.Context, _, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.key, nil
}

func TestGet(t *testing.T) {
	tests := []struct {
		name  string
		setup func(store *fakeStore)
		check func(t *testing.T, got dto.ScoringConfigView, err error)
	}{
		{
			name: "returns empty config when none saved",
			check: wantScoringConfig(dto.ScoringConfigView{
				Preferences:           dto.Preferences{Picks: []dto.Pick{}},
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedLocations:     []string{},
			}),
		},
		{
			name: "returns saved config",
			setup: func(store *fakeStore) {
				_, err := store.UpsertSearchConfig(context.Background(), dto.SearchConfig{
					UserID: "user-1", NotifyThreshold: 5,
				})
				if err != nil {
					t.Fatal(err)
				}
			},
			check: wantScoringConfig(dto.ScoringConfigView{
				NotifyThreshold:       5,
				Preferences:           dto.Preferences{Picks: []dto.Pick{}},
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedLocations:     []string{},
			}),
		},
		{
			name: "surfaces store error",
			setup: func(store *fakeStore) {
				store.GetErr = errors.New("db down")
			},
			check: func(t *testing.T, _ dto.ScoringConfigView, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("want error, got nil")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore().seedOptions()
			if tt.setup != nil {
				tt.setup(store)
			}
			svc := scoringconfig.New(store, &fakeReconsiderer{}, &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{}, &fakeBackfiller{})
			got, err := svc.Get(context.Background(), "user-1")
			tt.check(t, got, err)
		})
	}
}

func wantScoringConfig(want dto.ScoringConfigView) func(t *testing.T, got dto.ScoringConfigView, err error) {
	return func(t *testing.T, got dto.ScoringConfigView, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Get() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		in         dto.ScoringConfigView
		wantStatus int
	}{
		{
			name:       "rejects an unknown option",
			in:         dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:cobol", Stance: "nice"}}}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects a stance the dimension doesn't allow",
			in:         dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "block"}}}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects a notify threshold below 0",
			in:         dto.ScoringConfigView{NotifyThreshold: -1},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects a notify threshold above 100",
			in:         dto.ScoringConfigView{NotifyThreshold: 101},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects a negative salary floor amount",
			in:         dto.ScoringConfigView{Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: -1, Currency: "GBP"}}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects a salary floor with no currency",
			in:         dto.ScoringConfigView{Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: 55000}}},
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{}, &fakeBackfiller{})
			_, err := svc.Update(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestUpdateSucceeds(t *testing.T) {
	reconsiderer := &fakeReconsiderer{}
	recomputer := &fakeRecomputer{}
	backfiller := &fakeBackfiller{queued: 3}
	svc := scoringconfig.New(newFakeStore().seedOptions(), reconsiderer, recomputer, &fakeExtractor{}, &fakeCredentials{}, backfiller)

	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		NotifyThreshold:       70,
		ExcludedTitleKeywords: []string{" Intern ", ""},
		Preferences: dto.Preferences{
			Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "text"},
				{OptionID: "domain:gambling", Stance: "block"},
			},
			SalaryFloor: &dto.Money{Amount: 55000, Currency: "gbp"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"intern"}; len(got.ExcludedTitleKeywords) != 1 || got.ExcludedTitleKeywords[0] != want[0] {
		t.Fatalf("excluded title keywords = %v, want %v", got.ExcludedTitleKeywords, want)
	}
	wantPicks := []dto.Pick{
		{OptionID: "tech:go", Stance: "nice", Source: "manual"},
		{OptionID: "domain:gambling", Stance: "block", Source: "manual"},
	}
	if diff := cmp.Diff(wantPicks, got.Preferences.Picks); diff != "" {
		t.Errorf("picks mismatch, source forced to manual (-want +got):\n%s", diff)
	}
	wantFloor := &dto.Money{Amount: 55000, Currency: "GBP"}
	if diff := cmp.Diff(wantFloor, got.Preferences.SalaryFloor); diff != "" {
		t.Errorf("salary floor mismatch, currency uppercased (-want +got):\n%s", diff)
	}
	if reconsiderer.calledWith.UserID != "user-1" {
		t.Fatalf("reconsiderer called with %+v, want user-1", reconsiderer.calledWith)
	}
	if recomputer.calledWith != "user-1" || recomputer.calls != 1 {
		t.Fatalf("recomputer called %d times with %q, want once with user-1", recomputer.calls, recomputer.calledWith)
	}
	if backfiller.calledWith != "user-1" || backfiller.calls != 1 {
		t.Fatalf("backfiller called %d times with %q, want once with user-1", backfiller.calls, backfiller.calledWith)
	}
	if got.BackfillQueued != 3 {
		t.Fatalf("backfillQueued = %d, want 3", got.BackfillQueued)
	}
}

func TestUpdateBackfillFails(t *testing.T) {
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{}, &fakeBackfiller{err: errors.New("backfill blew up")})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestUpdateReconsiderFails(t *testing.T) {
	reconsiderer := &fakeReconsiderer{err: errors.New("reconsideration blew up")}
	svc := scoringconfig.New(newFakeStore().seedOptions(), reconsiderer, &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{}, &fakeBackfiller{})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if _, ok := apperr.StatusFor(err); ok {
		t.Fatalf("want unkinded error (mapped to 500 by the adapter), got a kinded one: %v", err)
	}
}

func TestUpdateRecomputeFails(t *testing.T) {
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{err: errors.New("recompute blew up")}, &fakeExtractor{}, &fakeCredentials{}, &fakeBackfiller{})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
