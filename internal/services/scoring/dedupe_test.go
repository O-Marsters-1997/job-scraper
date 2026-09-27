package scoring

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestEvaluatedPicksFor_ManualOverridesTextOnSameOption(t *testing.T) {
	byID := map[string]dto.ScoringOption{
		"tech:kubernetes": {ID: "tech:kubernetes", Dimension: dto.DimensionTech, Label: "Kubernetes", Question: "Does the role use Kubernetes?"},
	}
	answers := map[string]dto.Answer{
		questionHash("Does the role use Kubernetes?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	}
	picks := []dto.Pick{
		{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text"},
		{OptionID: "tech:kubernetes", Stance: "nice", Source: "manual"},
	}

	got := evaluatedPicksFor(picks, byID, answers)
	if len(got) != 1 {
		t.Fatalf("evaluatedPicksFor returned %d picks, want 1 (manual overrides text)", len(got))
	}
	if got[0].stance != "nice" {
		t.Fatalf("stance = %q, want %q (manual pick, not text)", got[0].stance, "nice")
	}
}
