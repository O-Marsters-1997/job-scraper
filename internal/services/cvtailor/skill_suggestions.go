package cvtailor

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

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
	return placeSkills(doc.Skills.Lines, bank, missing, answers), nil
}

func missingBankSkills(bank []dto.BankSkill, lines []docparse.SkillLine) []dto.BankSkill {
	var out []dto.BankSkill
	for _, b := range bank {
		if lineWith(lines, b.Name) < 0 {
			out = append(out, b)
		}
	}
	return out
}

func lineWith(lines []docparse.SkillLine, name string) int {
	if strings.TrimSpace(name) == "" {
		return -1
	}
	return slices.IndexFunc(lines, func(l docparse.SkillLine) bool {
		return slices.ContainsFunc(l.Items, func(item string) bool { return containsWord(item, name) })
	})
}

func containsWord(text, word string) bool {
	text, word = strings.ToLower(text), strings.ToLower(strings.TrimSpace(word))
	isWordRune := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	for from := 0; from <= len(text)-len(word); {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(word)
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		after, _ := utf8.DecodeRuneInString(text[end:])
		if (start == 0 || !isWordRune(before)) && (end == len(text) || !isWordRune(after)) {
			return true
		}
		from = start + 1
	}
	return false
}

func skillQuestions(missing []dto.BankSkill, lines []docparse.SkillLine) []string {
	var out []string
	seen := map[string]bool{}
	add := func(name string) {
		if q := skillQuestion(name); !seen[q] {
			seen[q] = true
			out = append(out, q)
		}
	}
	for _, b := range missing {
		add(b.Name)
	}
	for _, l := range lines {
		for _, item := range l.Items {
			add(item)
		}
	}
	return out
}

func skillLean(answers map[string]dto.Answer, name string) float64 {
	a := answers[skillQuestion(name)]
	return a.PYes - a.PNo
}

func placementLine(lines []docparse.SkillLine, bank []dto.BankSkill, b dto.BankSkill) int {
	if strings.TrimSpace(b.Category) == "" {
		return -1
	}
	for _, mate := range bank {
		if mate.ID != b.ID && strings.EqualFold(mate.Category, b.Category) {
			if i := lineWith(lines, mate.Name); i >= 0 {
				return i
			}
		}
	}
	return slices.IndexFunc(lines, func(l docparse.SkillLine) bool { return containsWord(l.Label, b.Category) })
}

func placeSkills(lines []docparse.SkillLine, bank, missing []dto.BankSkill, answers map[string]dto.Answer) dto.SkillSuggestions {
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
		i := placementLine(lines, bank, b)
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
		c := &lines[r.line].Candidates[r.cand]
		target, ok := lowestLean(lines[r.line].Base, replaced, r.line)
		if !ok || target.Score >= c.Score {
			continue
		}
		replaced[victim{r.line, target.Text}] = true
		c.Preselected, c.Replaces = true, target.Text
		picked++
	}
}

func lowestLean(base []dto.SkillItem, replaced map[victim]bool, line int) (dto.SkillItem, bool) {
	var best dto.SkillItem
	found := false
	for _, it := range base {
		if replaced[victim{line, it.Text}] {
			continue
		}
		if !found || it.Score < best.Score {
			best, found = it, true
		}
	}
	return best, found
}

type victim struct {
	line int
	text string
}
