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

func TestSuggestionsRankByPYesTimesConfidence(t *testing.T) {
	asker := &fakeAsker{answers: map[string]dto.Answer{
		questionPrefix + "a": {PYes: 0.9, Confidence: 0.2},
		questionPrefix + "b": {PYes: 0.5, Confidence: 0.9},
		questionPrefix + "c": {PYes: 0.1, Confidence: 1},
		questionPrefix + "d": {PYes: 0.7, Confidence: 0.7},
		questionPrefix + "e": {PYes: 0.6, Confidence: 0.5},
	}}
	svc, st := newService(t, nil, asker)
	p := addPosition(t, st, "Acme", "Engineer")
	addAchievements(t, p.ID, st, "a", "b", "c", "d", "e")

	got, err := svc.Suggestions(t.Context(), userID, dto.SuggestionsQuery{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got))
	selected := make([]bool, len(got))
	for i, s := range got {
		ids[i], selected[i] = s.Text, s.Preselected
	}
	if diff := cmp.Diff([]string{"d", "b", "e", "a", "c"}, ids); diff != "" {
		t.Errorf("Suggestions() order (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]bool{true, true, true, false, false}, selected); diff != "" {
		t.Errorf("Suggestions() preselected (-want +got):\n%s", diff)
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
