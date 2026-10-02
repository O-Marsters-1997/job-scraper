package cvtailor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

const questionPrefix = "Would this job value a candidate who: "

type fakeAsker struct {
	answers map[string]dto.Answer
	err     error
	asked   []string
}

func (f *fakeAsker) Ask(_ context.Context, _, _ string, questions []string) (map[string]dto.Answer, error) {
	f.asked = append(f.asked, questions...)
	if f.err != nil {
		return nil, f.err
	}
	return f.answers, nil
}

func addAchievements(t *testing.T, positionID string, st interface {
	CreateAchievement(context.Context, string, dto.AchievementInput) (dto.Achievement, error)
}, texts ...string) []dto.Achievement {
	t.Helper()
	var out []dto.Achievement
	for _, text := range texts {
		a, err := st.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: positionID, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	return out
}

func TestSuggestionsRankByNetLean(t *testing.T) {
	asker := &fakeAsker{answers: map[string]dto.Answer{
		questionPrefix + "strong":    {PYes: 0.7, PNo: 0.1, Confidence: 0.6},
		questionPrefix + "digest":    {PYes: 0.38, PNo: 0.36, Confidence: 0.1},
		questionPrefix + "e2e":       {PYes: 0.31, PNo: 0.34, Confidence: 0.05},
		questionPrefix + "microfend": {PYes: 0.13, PNo: 0.54, Confidence: 0.4},
	}}
	svc, st := newService(t, nil, asker)
	p := addPosition(t, st, "Acme", "Engineer")
	addAchievements(t, p.ID, st, "microfend", "e2e", "digest", "strong")

	got, err := svc.Suggestions(t.Context(), userID, dto.SuggestionsQuery{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		Text        string
		State       dto.SuggestionState
		Preselected bool
	}
	rows := make([]row, len(got))
	for i, s := range got {
		rows[i] = row{s.Text, s.State, s.Preselected}
	}
	want := []row{
		{"strong", dto.SuggestionFit, true},
		{"digest", dto.SuggestionFit, true},
		{"e2e", dto.SuggestionUnclear, true},
		{"microfend", dto.SuggestionLow, false},
	}
	if diff := cmp.Diff(want, rows); diff != "" {
		t.Errorf("Suggestions() (-want +got):\n%s", diff)
	}
}

func TestSuggestionsPreselectSlotCountOfMappedHeading(t *testing.T) {
	asker := &fakeAsker{answers: map[string]dto.Answer{}}
	docs := cvtailortest.Docs{TabJSON: cvTab(t, roleBlock{"Engineer, Acme", 2})}
	svc, st := newService(t, docs, asker)
	acme := addPosition(t, st, "Acme", "Engineer")
	other := addPosition(t, st, "Globex", "Engineer")
	addAchievements(t, acme.ID, st, "a1", "a2", "a3", "a4")
	addAchievements(t, other.ID, st, "g1", "g2", "g3", "g4")
	_, err := svc.SaveHeadings(t.Context(), userID, dto.HeadingMappingsInput{
		DocID: "doc", TabID: "tab", Mappings: []dto.HeadingMapping{{HeadingText: "Engineer, Acme", PositionID: &acme.ID}},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Suggestions(t.Context(), userID, dto.SuggestionsQuery{JobID: "job-1", DocID: "doc", TabID: "tab"})
	if err != nil {
		t.Fatal(err)
	}
	preselected := map[string]int{}
	for _, s := range got {
		if s.Preselected {
			preselected[s.PositionID]++
		}
	}
	if preselected[acme.ID] != 2 || preselected[other.ID] != 3 {
		t.Errorf("preselected per position = %v, want acme 2 (slot count), globex 3 (default)", preselected)
	}
}

func TestSuggestionsEmptyBankMakesNoAskCall(t *testing.T) {
	asker := &fakeAsker{}
	svc, _ := newService(t, nil, asker)
	got, err := svc.Suggestions(t.Context(), userID, dto.SuggestionsQuery{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 || len(asker.asked) != 0 {
		t.Errorf("Suggestions() = %v, asked %v; want none", got, asker.asked)
	}
}

func TestSuggestionsPropagatesMissingKey(t *testing.T) {
	missingKey := apperr.Unprocessable("connect an OpenRouter key")
	svc, st := newService(t, nil, &fakeAsker{err: missingKey})
	p := addPosition(t, st, "Acme", "Engineer")
	addAchievements(t, p.ID, st, "a")

	_, err := svc.Suggestions(t.Context(), userID, dto.SuggestionsQuery{JobID: "job-1"})
	if !errors.Is(err, missingKey) {
		t.Errorf("Suggestions() err = %v, want %v", err, missingKey)
	}
}
