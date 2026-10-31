package checks

import (
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	minHookWords = 25
	maxHookWords = 50
)

func Hook(text, company string) []dto.LetterFinding {
	findings := []dto.LetterFinding{}
	if words := len(strings.Fields(text)); words < minHookWords || words > maxHookWords {
		findings = append(findings, dto.LetterFinding{
			Kind: dto.FindingLength, Text: fmt.Sprintf("%d words, aim for %d to %d", words, minHookWords, maxHookWords),
		})
	}
	for _, phrase := range bannedPhrases(text) {
		findings = append(findings, dto.LetterFinding{Kind: dto.FindingAITell, Text: phrase})
	}
	if company != "" && !strings.Contains(strings.ToLower(text), strings.ToLower(company)) {
		findings = append(findings, dto.LetterFinding{Kind: dto.FindingMissingCompany, Text: company})
	}
	return findings
}
