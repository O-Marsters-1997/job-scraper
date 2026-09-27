package scoringconfig_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
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

type fakeBackfiller struct {
	err   error
	calls []string
}

func (f *fakeBackfiller) QueueUserBackfill(_ context.Context, userID string) (int64, error) {
	f.calls = append(f.calls, userID)
	return 0, f.err
}

func seededOptions() *providers.MockScoringOptionsProvider {
	options := providers.NewMockScoringOptionsProvider()
	options.Seed([]dto.ScoringOption{
		{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
		{ID: "domain:gambling", Dimension: dto.DimensionDomain, Label: "Gambling", Question: "Is the company's main business gambling?"},
		{ID: "seniority:senior", Dimension: dto.DimensionSeniority, Label: "Senior", Question: "Seniority?"},
	})
	return options
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
				Preferences:       dto.Preferences{Picks: []dto.Pick{}, BlockedTech: []string{}, Customs: []dto.CustomQuestion{}},
				ExcludedCompanies: []string{},
				ExcludedLocations: []string{},
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
				NotifyThreshold:   5,
				Preferences:       dto.Preferences{Picks: []dto.Pick{}, BlockedTech: []string{}, Customs: []dto.CustomQuestion{}},
				ExcludedCompanies: []string{},
				ExcludedLocations: []string{},
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
			svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{}, &fakeBackfiller{})
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
		{
			name: "rejects an 11th custom question",
			in: dto.ScoringConfigView{Preferences: dto.Preferences{Customs: func() []dto.CustomQuestion {
				customs := make([]dto.CustomQuestion, 11)
				for i := range customs {
					customs[i] = dto.CustomQuestion{Question: "Question?", Stance: "nice"}
				}
				return customs
			}()}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "rejects a custom question over 200 characters",
			in: dto.ScoringConfigView{Preferences: dto.Preferences{Customs: []dto.CustomQuestion{
				{Question: strings.Repeat("a", 201), Stance: "nice"},
			}}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "rejects a custom question with an unknown stance",
			in: dto.ScoringConfigView{Preferences: dto.Preferences{Customs: []dto.CustomQuestion{
				{Question: "Does the team pair program?", Stance: "block"},
			}}},
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSearchConfigProvider()
			svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{}, &fakeBackfiller{})
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
	backfiller := &fakeBackfiller{}
	svc := scoringconfig.New(store, reconsiderer, seededOptions(), recomputer, backfiller)

	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		NotifyThreshold: 70,
		Preferences: dto.Preferences{
			BlockedTech: []string{" Kubernetes ", ""},
			Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "text"},
				{OptionID: "domain:gambling", Stance: "block"},
			},
			SalaryFloor: &dto.Money{Amount: 55000, Currency: "gbp"},
			Customs: []dto.CustomQuestion{
				{Question: "  Does the team pair program?  ", Stance: "nice", Source: "text"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"kubernetes"}; len(got.Preferences.BlockedTech) != 1 || got.Preferences.BlockedTech[0] != want[0] {
		t.Fatalf("blocked tech = %v, want %v", got.Preferences.BlockedTech, want)
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
	wantCustoms := []dto.CustomQuestion{
		{Question: "Does the team pair program?", Stance: "nice", Source: "manual"},
	}
	if !reflect.DeepEqual(got.Preferences.Customs, wantCustoms) {
		t.Fatalf("customs = %+v, want %+v (trimmed, source forced to manual)", got.Preferences.Customs, wantCustoms)
	}
	if reconsiderer.calledWith.UserID != "user-1" {
		t.Fatalf("reconsiderer called with %+v, want user-1", reconsiderer.calledWith)
	}
	if recomputer.calledWith != "user-1" || recomputer.calls != 1 {
		t.Fatalf("recomputer called %d times with %q, want once with user-1", recomputer.calls, recomputer.calledWith)
	}
	if len(backfiller.calls) != 1 || backfiller.calls[0] != "user-1" {
		t.Fatalf("backfiller called %v, want one call with user-1 (a new custom question was added)", backfiller.calls)
	}
}

func TestUpdate_NoBackfillWhenNoCustomTextIsNew(t *testing.T) {
	store := providers.NewMockSearchConfigProvider()
	if _, err := store.UpsertSearchConfig(context.Background(), dto.SearchConfig{
		UserID: "user-1",
		Preferences: dto.Preferences{Customs: []dto.CustomQuestion{
			{Question: "Does the team pair program?", Stance: "nice", Source: "manual"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	backfiller := &fakeBackfiller{}
	svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{}, backfiller)

	if _, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{}); err != nil {
		t.Fatal(err)
	}
	if len(backfiller.calls) != 0 {
		t.Fatalf("backfiller called %v, want none (no custom text is new)", backfiller.calls)
	}
}

func TestUpdateReconsiderFails(t *testing.T) {
	store := providers.NewMockSearchConfigProvider()
	reconsiderer := &fakeReconsiderer{err: errors.New("reconsideration blew up")}
	svc := scoringconfig.New(store, reconsiderer, seededOptions(), &fakeRecomputer{}, &fakeBackfiller{})

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
	svc := scoringconfig.New(store, &fakeReconsiderer{}, seededOptions(), &fakeRecomputer{err: errors.New("recompute blew up")}, &fakeBackfiller{})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
