package cvtailor

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const maxPreselectedSwaps = 3

func skillQuestion(name string) string {
	return "Would this job value a candidate with experience of " + name + "?"
}

// SkillSuggestions places the user's Bank Skills missing from the base CV
// into its Skill Lines, preselecting the best fits as swaps.
func (s *Service) SkillSuggestions(ctx context.Context, userID string, q dto.SkillSuggestionsQuery) (dto.SkillSuggestions, error) {
	doc, err := loadTab(ctx, s.docs, userID, q.DocID, q.TabID)
	if err != nil {
		return dto.SkillSuggestions{}, err
	}
	empty := dto.SkillSuggestions{Lines: []dto.SkillLineSuggestion{}, Unplaced: []dto.SkillCandidate{}}
	if doc.Skills == nil {
		return empty, nil
	}
	bank, err := s.store.ListBankSkills(ctx, userID)
	if err != nil {
		return dto.SkillSuggestions{}, err
	}
	missing := missingBankSkills(bank, doc.Skills.Lines)
	if len(missing) == 0 {
		return empty, nil
	}
	answers, err := s.asker.Ask(ctx, userID, q.JobID, skillQuestions(missing, doc.Skills.Lines))
	if err != nil {
		return dto.SkillSuggestions{}, err
	}
	return placeSkills(doc.Skills.Lines, missing, answers), nil
}

func missingBankSkills(bank []dto.BankSkill, lines []docparse.SkillLine) []dto.BankSkill {
	have := map[string]bool{}
	for _, l := range lines {
		for _, item := range l.Items {
			have[strings.ToLower(item)] = true
		}
	}
	var out []dto.BankSkill
	for _, b := range bank {
		if !have[strings.ToLower(b.Name)] {
			out = append(out, b)
		}
	}
	return out
}

func skillQuestions(missing []dto.BankSkill, lines []docparse.SkillLine) []string {
	var out []string
	for _, b := range missing {
		out = append(out, skillQuestion(b.Name))
	}
	for _, l := range lines {
		for _, item := range l.Items {
			out = append(out, skillQuestion(item))
		}
	}
	return out
}

func skillLean(answers map[string]dto.Answer, name string) float64 {
	a := answers[skillQuestion(name)]
	return a.PYes - a.PNo
}

func placeSkills(lines []docparse.SkillLine, missing []dto.BankSkill, answers map[string]dto.Answer) dto.SkillSuggestions {
	out := dto.SkillSuggestions{Lines: make([]dto.SkillLineSuggestion, len(lines)), Unplaced: []dto.SkillCandidate{}}
	for i, l := range lines {
		out.Lines[i] = dto.SkillLineSuggestion{Label: l.Label, Base: make([]dto.SkillItem, len(l.Items)), Candidates: []dto.SkillCandidate{}}
		for j, item := range l.Items {
			lean := skillLean(answers, item)
			out.Lines[i].Base[j] = dto.SkillItem{Text: item, Score: lean, State: suggestionState(lean)}
		}
	}
	for _, b := range missing {
		lean := skillLean(answers, b.Name)
		c := dto.SkillCandidate{BankSkillID: b.ID, Name: b.Name, Score: lean, State: suggestionState(lean)}
		i := slices.IndexFunc(lines, func(l docparse.SkillLine) bool { return strings.EqualFold(l.Label, b.Category) })
		if i < 0 {
			out.Unplaced = append(out.Unplaced, c)
			continue
		}
		out.Lines[i].Candidates = append(out.Lines[i].Candidates, c)
	}
	byLean := func(a, b dto.SkillCandidate) int { return cmp.Compare(b.Score, a.Score) }
	slices.SortStableFunc(out.Unplaced, byLean)
	for i := range out.Lines {
		slices.SortStableFunc(out.Lines[i].Candidates, byLean)
	}
	preselectSwaps(out.Lines)
	return out
}

func preselectSwaps(lines []dto.SkillLineSuggestion) {
	type ref struct{ line, cand int }
	var fits []ref
	for i, l := range lines {
		for j, c := range l.Candidates {
			if c.State == dto.SuggestionFit {
				fits = append(fits, ref{i, j})
			}
		}
	}
	slices.SortStableFunc(fits, func(a, b ref) int {
		return cmp.Compare(lines[b.line].Candidates[b.cand].Score, lines[a.line].Candidates[a.cand].Score)
	})
	replaced := map[victim]bool{}
	picked := 0
	for _, r := range fits {
		if picked == maxPreselectedSwaps {
			return
		}
		victim, ok := lowestLean(lines[r.line].Base, replaced, r.line)
		if !ok {
			continue
		}
		c := &lines[r.line].Candidates[r.cand]
		c.Preselected, c.Replaces = true, victim
		picked++
	}
}

func lowestLean(base []dto.SkillItem, replaced map[victim]bool, line int) (string, bool) {
	best := -1
	for i, it := range base {
		if replaced[victim{line, it.Text}] {
			continue
		}
		if best < 0 || it.Score < base[best].Score {
			best = i
		}
	}
	if best < 0 {
		return "", false
	}
	replaced[victim{line, base[best].Text}] = true
	return base[best].Text, true
}

type victim struct {
	line int
	text string
}
