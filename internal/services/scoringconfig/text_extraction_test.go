package scoringconfig_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

func TestUpdate_UnchangedTextSkipsExtraction(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}}
	svc := scoringconfig.New(newFakeStore(), scoringtest.Reconsiders(), scoringtest.Recomputes(0), extractor, &fakeCredentials{key: "sk-test"}, scoringtest.Backfills(0))

	in := dto.ScoringConfigView{Preferences: dto.Preferences{PreferenceText: "I want to work with Go"}}

	if _, err := svc.Update(context.Background(), "user-1", in); err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 {
		t.Fatalf("extractor called %d times, want 1 (first save with text)", extractor.calls)
	}

	if _, err := svc.Update(context.Background(), "user-1", in); err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 {
		t.Fatalf("extractor called %d times, want 1 (unchanged text)", extractor.calls)
	}
}

func TestUpdate_ReextractionReplacesTextPicksKeepsManual(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}}
	svc := scoringconfig.New(newFakeStore(), scoringtest.Reconsiders(), scoringtest.Recomputes(0), extractor, &fakeCredentials{key: "sk-test"}, scoringtest.Backfills(0))

	manualPick := dto.Pick{OptionID: "seniority:senior", Stance: "nice"}
	if _, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{Picks: []dto.Pick{manualPick}, PreferenceText: "I know Go"},
	}); err != nil {
		t.Fatal(err)
	}

	extractor.picks = []dto.Pick{{OptionID: "domain:gambling", Stance: "avoid", Source: "text"}}
	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{Picks: []dto.Pick{manualPick}, PreferenceText: "avoid gambling companies"},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPicks := []dto.Pick{
		{OptionID: "seniority:senior", Stance: "nice", Source: "manual"},
		{OptionID: "domain:gambling", Stance: "avoid", Source: "text"},
	}
	if diff := cmp.Diff(wantPicks, got.Preferences.Picks); diff != "" {
		t.Errorf("picks mismatch, re-extraction replaces text picks, keeps manual (-want +got):\n%s", diff)
	}
}

func TestUpdate_ManualOverridesText(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text"}}}
	svc := scoringconfig.New(newFakeStore(), scoringtest.Reconsiders(), scoringtest.Recomputes(0), extractor, &fakeCredentials{key: "sk-test"}, scoringtest.Backfills(0))

	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{
			Picks:          []dto.Pick{{OptionID: "tech:kubernetes", Stance: "nice"}},
			PreferenceText: "avoid Kubernetes",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPicks := []dto.Pick{
		{OptionID: "tech:kubernetes", Stance: "nice", Source: "manual"},
		{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text", Overridden: true},
	}
	if diff := cmp.Diff(wantPicks, got.Preferences.Picks); diff != "" {
		t.Errorf("picks mismatch, manual wins, text pick shows overridden (-want +got):\n%s", diff)
	}
}

func TestUpdate_DropsHallucinatedOptionID(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{
		{OptionID: "tech:go", Stance: "nice", Source: "text"},
		{OptionID: "tech:made-up", Stance: "nice", Source: "text"},
	}}
	svc := scoringconfig.New(newFakeStore(), scoringtest.Reconsiders(), scoringtest.Recomputes(0), extractor, &fakeCredentials{key: "sk-test"}, scoringtest.Backfills(0))

	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{PreferenceText: "I like Go, and made-up-thing"},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPicks := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}
	if diff := cmp.Diff(wantPicks, got.Preferences.Picks); diff != "" {
		t.Errorf("picks mismatch, hallucinated id dropped, not rejected (-want +got):\n%s", diff)
	}
}

func TestUpdate_NoCredentialReturnsUnprocessable(t *testing.T) {
	svc := scoringconfig.New(newFakeStore(), scoringtest.Reconsiders(), scoringtest.Recomputes(0), &fakeExtractor{}, &fakeCredentials{err: errors.New("no credential")}, scoringtest.Backfills(0))

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{PreferenceText: "I like Go"},
	})
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusUnprocessableEntity)
	}
}
