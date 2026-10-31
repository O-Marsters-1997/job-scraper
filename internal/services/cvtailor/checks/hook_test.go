package checks_test

import (
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

func hookOf(words int, extra string) string {
	return "Acme's platform work caught my eye. " + extra + strings.Repeat("filler ", words)
}

func TestHook(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		text string
		want dto.LetterFindingKind
	}{
		{"too short", "Acme platform work caught my eye.", dto.FindingLength},
		{"too long", hookOf(60, ""), dto.FindingLength},
		{"ai tell", hookOf(25, "I leverage things. "), dto.FindingAITell},
		{"missing company", strings.ReplaceAll(hookOf(25, ""), "Acme", "Your"), dto.FindingMissingCompany},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checks.Hook(tc.text, "Acme")
			for _, f := range got {
				if f.Kind == tc.want {
					return
				}
			}
			t.Errorf("Hook() = %+v, want kind %q", got, tc.want)
		})
	}

	t.Run("a clean hook has no findings, not nil", func(t *testing.T) {
		t.Parallel()
		got := checks.Hook(hookOf(25, ""), "ACME")
		if got == nil || len(got) != 0 {
			t.Errorf("Hook() = %#v, want empty non-nil", got)
		}
	})
}
