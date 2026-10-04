package scoring_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

var errBoom = errors.New("boom")

type erroringGetStore struct{ *scoringtest.FakeStore }

func (s *erroringGetStore) GetSearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, errBoom
}

type failingQueueStore struct{ *scoringtest.FakeStore }

func (s *failingQueueStore) QueueMissingAnswers(context.Context, string, []string, string) (int64, error) {
	return 0, errBoom
}

type failingInputsStore struct{ *scoringtest.FakeStore }

func (s *failingInputsStore) ListScoringInputs(context.Context, string, string) ([]store.ScoringInput, error) {
	return nil, errBoom
}

func TestGet(t *testing.T) {
	empty := dto.ScoringConfigView{
		Preferences:           dto.Preferences{Picks: []dto.Pick{}},
		ExcludedTitleKeywords: []string{},
		ExcludedCompanies:     []string{},
		ExcludedLocations:     []string{},
		RequiredLocations:     []string{},
		RequiredTitleKeywords: []string{},
	}
	saved := empty
	saved.NotifyThreshold = 5

	tests := []struct {
		name string
		seed *dto.SearchConfig
		want dto.ScoringConfigView
	}{
		{"returns empty config when none saved", nil, empty},
		{"returns saved config", &dto.SearchConfig{UserID: "user-1", NotifyThreshold: 5}, saved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFakeStore()
			if tt.seed != nil {
				st.SeedSearchConfig(*tt.seed)
			}
			got, err := newService(t, st).GetConfig(t.Context(), "user-1")
			if err != nil {
				t.Fatalf("GetConfig() err = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetConfig() (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetErrors(t *testing.T) {
	svc := newService(t, &erroringGetStore{newFakeStore()})
	if _, err := svc.GetConfig(t.Context(), "user-1"); !errors.Is(err, errBoom) {
		t.Errorf("GetConfig() err = %v, want boom", err)
	}
}

func TestUpdateConfigRejects(t *testing.T) {
	tests := []struct {
		name string
		in   dto.ScoringConfigView
	}{
		{"an unknown option", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:cobol", Stance: "nice"}}}}},
		{"a stance the dimension doesn't allow", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "block"}}}}},
		{"a ladder pick without a weight", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "seniority:senior", Stance: "nice"}}}}},
		{"a ladder weight above 100", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "seniority:senior", Stance: "nice", Weight: 101}}}}},
		{"a ladder pick with an ok stance", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "seniority:senior", Stance: "ok", Weight: 50}}}}},
		{"a weight on a non-ladder pick", dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Weight: 50}}}}},
		{"a notify threshold below 0", dto.ScoringConfigView{NotifyThreshold: -1}},
		{"a notify threshold above 100", dto.ScoringConfigView{NotifyThreshold: 101}},
		{"a negative salary floor amount", dto.ScoringConfigView{Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: -1, Currency: "GBP"}}}},
		{"a salary floor with no currency", dto.ScoringConfigView{Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: 55000}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newService(t, newFakeStore()).UpdateConfig(t.Context(), "user-1", tt.in)
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("UpdateConfig() err = %v, want an invalid apperr", err)
			}
		})
	}
}

func TestUpdateConfig(t *testing.T) {
	svc := newService(t, newFakeStore())

	got, err := svc.UpdateConfig(t.Context(), "user-1", dto.ScoringConfigView{
		NotifyThreshold:       70,
		ExcludedTitleKeywords: []string{" Intern ", ""},
		Preferences: dto.Preferences{
			Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "text"},
				{OptionID: "domain:gambling", Stance: "block"},
				{OptionID: "tech:kubernetes", Stance: "ok"},
				{OptionID: "seniority:senior", Stance: "nice", Weight: 70},
			},
			SalaryFloor: &dto.Money{Amount: 55000, Currency: "gbp"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateConfig() err = %v", err)
	}
	if diff := cmp.Diff([]string{"intern"}, got.ExcludedTitleKeywords); diff != "" {
		t.Errorf("excluded title keywords, trimmed and lowercased (-want +got):\n%s", diff)
	}
	wantPicks := []dto.Pick{
		{OptionID: "tech:go", Stance: "nice", Source: "manual"},
		{OptionID: "domain:gambling", Stance: "block", Source: "manual"},
		{OptionID: "tech:kubernetes", Stance: "ok", Source: "manual"},
		{OptionID: "seniority:senior", Stance: "nice", Weight: 70, Source: "manual"},
	}
	if diff := cmp.Diff(wantPicks, got.Preferences.Picks); diff != "" {
		t.Errorf("picks, source forced to manual (-want +got):\n%s", diff)
	}
	wantFloor := &dto.Money{Amount: 55000, Currency: "GBP"}
	if diff := cmp.Diff(wantFloor, got.Preferences.SalaryFloor); diff != "" {
		t.Errorf("salary floor, currency uppercased (-want +got):\n%s", diff)
	}
	if got.BackfillQueued != 4 {
		t.Errorf("BackfillQueued = %d, want 4", got.BackfillQueued)
	}
}

func TestUpdateConfigFailures(t *testing.T) {
	withGo := dto.ScoringConfigView{Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice"}}}}
	tests := []struct {
		name string
		st   scoring.Store
		opts []depsOpt
		in   dto.ScoringConfigView
	}{
		{"backfill fails", &failingQueueStore{newFakeStore()}, nil, withGo},
		{"reconsider fails", newFakeStore(), []depsOpt{withCandidates(scoringtest.ReconsiderFails(errBoom))}, dto.ScoringConfigView{}},
		{"recompute fails", &failingInputsStore{newFakeStore()}, nil, withGo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newService(t, tt.st, tt.opts...).UpdateConfig(t.Context(), "user-1", tt.in)
			if !errors.Is(err, errBoom) {
				t.Fatalf("UpdateConfig() err = %v, want boom", err)
			}
			if _, ok := errors.AsType[*apperr.Error](err); ok {
				t.Errorf("UpdateConfig() err = %v, want an unkinded error (the adapter maps it to 500)", err)
			}
		})
	}
}
