package scoringconfig_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

type fakeReconsiderer struct {
	err        error
	calledWith dto.SearchConfig
}

func (f *fakeReconsiderer) Reconsider(_ context.Context, config dto.SearchConfig) error {
	f.calledWith = config
	return f.err
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
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedSeniority:     []string{},
				ExcludedLocations:     []string{},
				ScoringQuestions: dto.ScoringQuestions{
					Criteria: []dto.ScoringCriterion{},
					Scale:    []string{},
				},
			},
		},
		{
			name: "returns saved config",
			setup: func(store *providers.MockSearchConfigProvider) {
				_, err := store.UpsertSearchConfig(context.Background(), dto.SearchConfig{
					UserID:            "user-1",
					SuitabilityRubric: "senior go",
					NotifyThreshold:   5,
				})
				if err != nil {
					t.Fatal(err)
				}
			},
			want: dto.ScoringConfigView{
				SuitabilityRubric:     "senior go",
				NotifyThreshold:       5,
				ExcludedTitleKeywords: []string{},
				ExcludedCompanies:     []string{},
				ExcludedSeniority:     []string{},
				ExcludedLocations:     []string{},
				ScoringQuestions: dto.ScoringQuestions{
					Criteria: []dto.ScoringCriterion{},
					Scale:    []string{},
				},
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
			svc := scoringconfig.New(store, &fakeReconsiderer{}, nil)
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
			name:       "rejects unknown seniority level",
			in:         dto.ScoringConfigView{ExcludedSeniority: []string{"wizard"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "rejects scale with fewer than 2 levels",
			in: dto.ScoringConfigView{
				ScoringQuestions: dto.ScoringQuestions{
					Profile: "senior go",
					Scale:   []string{"Only level"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "rejects empty profile with no criteria",
			in: dto.ScoringConfigView{
				ScoringQuestions: dto.ScoringQuestions{
					Scale: []string{"Low", "High"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "rejects duplicate criterion keys",
			in: dto.ScoringConfigView{
				ScoringQuestions: dto.ScoringQuestions{
					Profile: "senior go",
					Criteria: []dto.ScoringCriterion{
						{Key: "go_backend"},
						{Key: "go_backend"},
					},
					Scale: []string{"Low", "High"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockSearchConfigProvider()
			svc := scoringconfig.New(store, &fakeReconsiderer{}, nil)
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
	svc := scoringconfig.New(store, reconsiderer, nil)

	questions := dto.ScoringQuestions{
		Profile: "Senior Go engineer",
		Criteria: []dto.ScoringCriterion{
			{Key: "go_backend", Instructions: "Does the job use Go?", True: "yes", False: "no", Required: true},
		},
		Scale: []string{"Not relevant", "Weak", "Possible", "Strong", "Apply today"},
	}
	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		SuitabilityRubric:     "  Senior Go  ",
		NotifyThreshold:       7,
		ExcludedTitleKeywords: []string{" Intern ", ""},
		ExcludedSeniority:     []string{"Junior"},
		ScoringQuestions:      questions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.SuitabilityRubric != "  Senior Go  " {
		t.Fatalf("rubric = %q, want unchanged", got.SuitabilityRubric)
	}
	if want := []string{"intern"}; len(got.ExcludedTitleKeywords) != 1 || got.ExcludedTitleKeywords[0] != want[0] {
		t.Fatalf("excluded title keywords = %v, want %v", got.ExcludedTitleKeywords, want)
	}
	if want := []string{"junior"}; len(got.ExcludedSeniority) != 1 || got.ExcludedSeniority[0] != want[0] {
		t.Fatalf("excluded seniority = %v, want %v", got.ExcludedSeniority, want)
	}
	if !reflect.DeepEqual(got.ScoringQuestions, questions) {
		t.Fatalf("scoring questions = %+v, want %+v", got.ScoringQuestions, questions)
	}
	if reconsiderer.calledWith.UserID != "user-1" {
		t.Fatalf("reconsiderer called with %+v, want user-1", reconsiderer.calledWith)
	}
}

func TestUpdateReconsiderFails(t *testing.T) {
	store := providers.NewMockSearchConfigProvider()
	reconsiderer := &fakeReconsiderer{err: errors.New("reconsideration blew up")}
	svc := scoringconfig.New(store, reconsiderer, nil)

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		ScoringQuestions: dto.ScoringQuestions{
			Profile: "senior go",
			Scale:   []string{"Low", "High"},
		},
	})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if _, ok := apperr.StatusFor(err); ok {
		t.Fatalf("want unkinded error (mapped to 500 by the adapter), got a kinded one: %v", err)
	}
}

type fakeRescorer struct {
	queued int64
	err    error
}

func (f *fakeRescorer) QueueRescore(context.Context, string) (int64, error) {
	return f.queued, f.err
}

func TestRescore(t *testing.T) {
	svc := scoringconfig.New(providers.NewMockSearchConfigProvider(), &fakeReconsiderer{}, &fakeRescorer{queued: 3})
	got, err := svc.Rescore(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Queued != 3 {
		t.Fatalf("queued = %d, want 3", got.Queued)
	}
}
