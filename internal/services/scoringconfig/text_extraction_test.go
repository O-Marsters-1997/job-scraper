package scoringconfig_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

func TestUpdate_UnchangedTextSkipsExtraction(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}}
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, extractor, &fakeCredentials{key: "sk-test"})

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
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, extractor, &fakeCredentials{key: "sk-test"})

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
	if !reflect.DeepEqual(got.Preferences.Picks, wantPicks) {
		t.Fatalf("picks = %+v, want %+v (re-extraction replaces text picks, keeps manual)", got.Preferences.Picks, wantPicks)
	}
}

func TestUpdate_ManualOverridesText(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text"}}}
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, extractor, &fakeCredentials{key: "sk-test"})

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
	if !reflect.DeepEqual(got.Preferences.Picks, wantPicks) {
		t.Fatalf("picks = %+v, want %+v (manual wins, text pick shows overridden)", got.Preferences.Picks, wantPicks)
	}
}

func TestUpdate_DropsHallucinatedOptionID(t *testing.T) {
	extractor := &fakeExtractor{picks: []dto.Pick{
		{OptionID: "tech:go", Stance: "nice", Source: "text"},
		{OptionID: "tech:made-up", Stance: "nice", Source: "text"},
	}}
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, extractor, &fakeCredentials{key: "sk-test"})

	got, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{PreferenceText: "I like Go, and made-up-thing"},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPicks := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}
	if !reflect.DeepEqual(got.Preferences.Picks, wantPicks) {
		t.Fatalf("picks = %+v, want %+v (hallucinated id dropped, not rejected)", got.Preferences.Picks, wantPicks)
	}
}

func TestUpdate_NoCredentialReturnsUnprocessable(t *testing.T) {
	svc := scoringconfig.New(newFakeStore().seedOptions(), &fakeReconsiderer{}, &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{err: errors.New("no credential")})

	_, err := svc.Update(context.Background(), "user-1", dto.ScoringConfigView{
		Preferences: dto.Preferences{PreferenceText: "I like Go"},
	})
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusUnprocessableEntity)
	}
}
