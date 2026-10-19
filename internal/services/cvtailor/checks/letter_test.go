package checks_test

import (
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

func letterOf(n int, extra string) string {
	return "I am writing to apply to Acme as a Platform Engineer. " + extra + strings.Repeat("filler ", n)
}

func TestLetter(t *testing.T) {
	t.Parallel()
	achievements := []checks.LetterAchievement{{ID: "a1", Text: "Cut deploy latency by forty percent across services"}}
	clean := letterOf(300, "")

	for _, tc := range []struct {
		name string
		text string
		want dto.LetterFindingKind
	}{
		{"leftover bullet", letterOf(300, "\n• Cut latency\n"), dto.FindingLeftoverBullet},
		{"leftover underscore gap", letterOf(300, "\nI know ___ well.\n"), dto.FindingLeftoverGap},
		{"leftover scaffold gap", letterOf(300, "\n[GAP] Kubernetes\n"), dto.FindingLeftoverGap},
		{"too short", letterOf(50, ""), dto.FindingLength},
		{"too long", letterOf(450, ""), dto.FindingLength},
		{"ai tell", letterOf(300, "I leverage my skills."), dto.FindingAITell},
		{"restates achievement", letterOf(300, "I cut deploy latency by forty percent across services."), dto.FindingRestatesAchievement},
		{"missing company", strings.ReplaceAll(clean, "Acme", "them"), dto.FindingMissingCompany},
		{"missing role", strings.ReplaceAll(clean, "Platform Engineer", "builder"), dto.FindingMissingRole},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checks.Letter(tc.text, "Acme", "Platform Engineer", achievements)
			for _, f := range got.Findings {
				if f.Kind != tc.want {
					continue
				}
				if tc.want == dto.FindingRestatesAchievement && f.AchievementID != "a1" {
					t.Errorf("finding = %+v, want achievement a1", f)
				}
				return
			}
			t.Errorf("Letter() findings = %+v, want kind %q", got.Findings, tc.want)
		})
	}

	t.Run("clean letter has no findings", func(t *testing.T) {
		t.Parallel()
		got := checks.Letter(clean, "Acme", "Platform Engineer", achievements)
		if got.Words < 250 || got.Words > 400 || len(got.Findings) != 0 {
			t.Errorf("Letter() = %+v, want 250-400 words and no findings", got)
		}
	})
}
