package scoring_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func TestUpdate_UnchangedTextSkipsExtraction(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}}
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: extractor})

	in := dto.ScoringConfigView{Preferences: dto.Preferences{PreferenceText: "I want to work with Go"}}

	if _, err := svc.UpdateConfig(context.Background(), "user-1", in); err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 {
		t.Fatalf("extractor called %d times, want 1 (first save with text)", extractor.calls)
	}

	if _, err := svc.UpdateConfig(context.Background(), "user-1", in); err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 {
		t.Fatalf("extractor called %d times, want 1 (unchanged text)", extractor.calls)
	}
}

func TestUpdate_ReextractionReplacesTextPicksKeepsManual(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}}
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: extractor})

	manualPick := dto.Pick{OptionID: "seniority:senior", Stance: "nice"}
	if _, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{Picks: []dto.Pick{manualPick}, PreferenceText: "I know Go"},
	}); err != nil {
		t.Fatal(err)
	}

	extractor.picks = []dto.Pick{{OptionID: "domain:gambling", Stance: "avoid", Source: "text"}}
	got, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{
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
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: extractor})

	got, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{
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
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: extractor})

	got, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{
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
	svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, Candidates: scoringtest.Reconsiders(), Extractor: &fakeExtractor{}})

	_, err := svc.UpdateConfig(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{PreferenceText: "I like Go"},
	})
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusUnprocessableEntity)
	}
}
