package scoring_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

type erroringGetStore struct {
	*scoringtest.FakeStore
	err error
}

func (s *erroringGetStore) GetSearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, s.err
}

type fakeExtractor struct {
	calls int
	picks []dto.Pick
	err   error
}

func (f *fakeExtractor) Extract(context.Context, string, string, []dto.ScoringOption, []dto.DimensionSpec) ([]dto.Pick, error) {
	f.calls++
	return f.picks, f.err
}

type failingQueueStore struct {
	*scoringtest.FakeStore
	err error
}

func (s *failingQueueStore) QueueMissingAnswers(context.Context, string, []string, string) (int64, error) {
	return 0, s.err
}

type failingInputsStore struct {
	*scoringtest.FakeStore
	err error
}

func (s *failingInputsStore) ListScoringInputs(context.Context, string, string) ([]store.ScoringInput, error) {
	return nil, s.err
}

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		newStore func(t *testing.T) scoring.Store
		check    func(t *testing.T, got dto.ScoringConfigView, err error)
	}{
		{
			name:     "returns empty config when none saved",
			newStore: func(*testing.T) scoring.Store { return newFakeStore() },
			check: wantScoringConfig(dto.ScoringConfigView{
				Preferences:           dto.Preferences{Picks: []dto.Pick{}},
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedLocations:     []string{},
			}),
		},
		{
			name: "returns saved config",
			newStore: func(t *testing.T) scoring.Store {
				t.Helper()
				st := newFakeStore()
				if _, err := st.UpsertSearchConfig(context.Background(), dto.SearchConfig{UserID: "user-1", NotifyThreshold: 5}); err != nil {
					t.Fatal(err)
				}
				return st
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
			newStore: func(*testing.T) scoring.Store {
				return &erroringGetStore{FakeStore: newFakeStore(), err: errors.New("db down")}
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
			svc := scoring.NewService(scoring.Deps{Store: tt.newStore(t), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: &fakeExtractor{}})
			got, err := svc.GetConfig(context.Background(), "user-1")
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
			svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: &fakeExtractor{}})
			_, err := svc.UpdateConfig(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestUpdateSucceeds(t *testing.T) {
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: &fakeExtractor{}})

	got, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{
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
	if got.BackfillQueued != 2 {
		t.Fatalf("backfillQueued = %d, want 2", got.BackfillQueued)
	}
}

func TestUpdateBackfillFails(t *testing.T) {
	svc := scoring.NewService(scoring.Deps{Store: &failingQueueStore{FakeStore: newFakeStore(), err: errors.New("backfill blew up")}, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: &fakeExtractor{}})

	_, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice"}}}})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestUpdateReconsiderFails(t *testing.T) {
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.ReconsiderFails(errors.New("reconsideration blew up")), Extractor: &fakeExtractor{}})

	_, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if _, ok := apperr.StatusFor(err); ok {
		t.Fatalf("want unkinded error (mapped to 500 by the adapter), got a kinded one: %v", err)
	}
}

func TestUpdateRecomputeFails(t *testing.T) {
	svc := scoring.NewService(scoring.Deps{Store: &failingInputsStore{FakeStore: newFakeStore(), err: errors.New("recompute blew up")}, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: &fakeExtractor{}})

	_, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice"}}}})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
