package scoringconfig_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/services/scoringconfig"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

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

func seededOptions() *providers.MockScoringOptionsProvider {
	options := providers.NewMockScoringOptionsProvider()
	options.Seed([]dto.ScoringOption{
		{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
		{ID: "tech:kubernetes", Dimension: dto.DimensionTech, Label: "Kubernetes", Question: "Does the role use Kubernetes?"},
		{ID: "domain:gambling", Dimension: dto.DimensionDomain, Label: "Gambling", Question: "Is the company's main business gambling?"},
		{ID: "seniority:senior", Dimension: dto.DimensionSeniority, Label: "Senior", Question: "Seniority?"},
	})
	return options
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
		name    string
		setup   func(store *providers.MockSearchConfigProvider)
		want    dto.ScoringConfigView
		wantErr bool
	}{
		{
			name: "returns empty config when none saved",
			want: dto.ScoringConfigView{
				Preferences:           dto.Preferences{Picks: []dto.Pick{}},
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedLocations:     []string{},
			},
		},
		{
			name: "returns saved config",
			setup: func(store *providers.MockSearchConfigProvider) {
				_, err := store.UpsertSearchConfig(context.Background(), dto.SearchConfig{
					UserID: "user-1", NotifyThreshold: 5,
				})
				if err != nil {
					t.Fatal(err)
				}
			},
			want: dto.ScoringConfigView{
				NotifyThreshold:       5,
				Preferences:           dto.Preferences{Picks: []dto.Pick{}},
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedLocations:     []string{},
			},
		},
		{
			name: "surfaces non-not-found error",
			setup: func(store *providers.MockSearchConfigProvider) {
				store.GetErr = errors.New("db down")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSearchConfigProvider()
			if tt.setup != nil {
				tt.setup(store)
			}
			svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{})
			got, err := svc.Get(context.Background(), "user-1")
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
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
			store := providers.NewMockSearchConfigProvider()
			svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{})
			_, err := svc.Update(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestUpdateSucceeds(t *testing.T) {
	store := providers.NewMockSearchConfigProvider()
	reconsiderer := &fakeReconsiderer{}
	recomputer := &fakeRecomputer{}
	svc := scoringconfig.New(store, reconsiderer, seededOptions(), recomputer, &fakeExtractor{}, &fakeCredentials{})

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
	if !reflect.DeepEqual(got.Preferences.Picks, wantPicks) {
		t.Fatalf("picks = %+v, want %+v (source forced to manual)", got.Preferences.Picks, wantPicks)
	}
	wantFloor := &dto.Money{Amount: 55000, Currency: "GBP"}
	if !reflect.DeepEqual(got.Preferences.SalaryFloor, wantFloor) {
		t.Fatalf("salary floor = %+v, want %+v (currency uppercased)", got.Preferences.SalaryFloor, wantFloor)
	}
	if reconsiderer.calledWith.UserID != "user-1" {
		t.Fatalf("reconsiderer called with %+v, want user-1", reconsiderer.calledWith)
	}
	if recomputer.calledWith != "user-1" || recomputer.calls != 1 {
		t.Fatalf("recomputer called %d times with %q, want once with user-1", recomputer.calls, recomputer.calledWith)
	}
}

func TestUpdateReconsiderFails(t *testing.T) {
	store := providers.NewMockSearchConfigProvider()
	reconsiderer := &fakeReconsiderer{err: errors.New("reconsideration blew up")}
	svc := scoringconfig.New(store, reconsiderer, seededOptions(), &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if _, ok := apperr.StatusFor(err); ok {
		t.Fatalf("want unkinded error (mapped to 500 by the adapter), got a kinded one: %v", err)
	}
}

func TestUpdateRecomputeFails(t *testing.T) {
	store := providers.NewMockSearchConfigProvider()
	svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{err: errors.New("recompute blew up")}, &fakeExtractor{}, &fakeCredentials{})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
