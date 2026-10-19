package checks

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	minLetterWords = 250
	maxLetterWords = 400

	minRestatedWords = 5
	restatedOverlap  = 0.8
)

var (
	gapRe       = regexp.MustCompile(`_{3,}|\[GAP\]`)
	plainWordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)
	sentenceRe  = regexp.MustCompile(`[.!?]+\s+|\n`)
)

type LetterAchievement struct {
	ID   string
	Text string
}

func Letter(text, company, role string, achievements []LetterAchievement) dto.LetterCheck {
	words := len(strings.Fields(text))
	findings := []dto.LetterFinding{}
	add := func(kind dto.LetterFindingKind, text, achievementID string) {
		findings = append(findings, dto.LetterFinding{Kind: kind, Text: text, AchievementID: achievementID})
	}

	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "•"):
			add(dto.FindingLeftoverBullet, line, "")
		case gapRe.MatchString(line):
			add(dto.FindingLeftoverGap, line, "")
		}
	}
	if words < minLetterWords || words > maxLetterWords {
		add(dto.FindingLength, fmt.Sprintf("%d words, aim for %d to %d", words, minLetterWords, maxLetterWords), "")
	}
	for _, phrase := range bannedPhrases(text) {
		add(dto.FindingAITell, phrase, "")
	}
	for _, a := range achievements {
		if s, ok := restatingSentence(text, a.Text); ok {
			add(dto.FindingRestatesAchievement, s, a.ID)
		}
	}
	if company != "" && !strings.Contains(strings.ToLower(text), strings.ToLower(company)) {
		add(dto.FindingMissingCompany, company, "")
	}
	if role != "" && !namesRole(text, role) {
		add(dto.FindingMissingRole, role, "")
	}
	return dto.LetterCheck{Words: words, Findings: findings}
}

func restatingSentence(text, achievement string) (string, bool) {
	want := wordSet(achievement)
	if len(want) < minRestatedWords {
		return "", false
	}
	for _, s := range sentenceRe.Split(text, -1) {
		have := wordSet(s)
		shared := 0
		for w := range want {
			if have[w] {
				shared++
			}
		}
		if float64(shared) >= restatedOverlap*float64(len(want)) {
			return strings.TrimSpace(s), true
		}
	}
	return "", false
}

func namesRole(text, role string) bool {
	have := wordSet(text)
	var total, found int
	for w := range wordSet(role) {
		if len(w) < 3 {
			continue
		}
		total++
		if have[w] {
			found++
		}
	}
	return total == 0 || found*2 >= total
}

func wordSet(s string) map[string]bool {
	set := map[string]bool{}
	for _, w := range plainWordRe.FindAllString(strings.ToLower(s), -1) {
		set[w] = true
	}
	return set
}
