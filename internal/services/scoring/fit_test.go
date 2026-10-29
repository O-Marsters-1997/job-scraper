package scoring

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestComputeUsesParams(t *testing.T) {
	pick := evaluatedPick{dimension: dto.DimensionTech, key: "tech:go", stance: "nice", answer: dto.Answer{PYes: 1}, known: true}
	miss := evaluatedPick{dimension: dto.DimensionRole, key: "role:backend", stance: "nice", answer: dto.Answer{PNo: 1}, known: true}
	picks := []evaluatedPick{pick, miss}

	base, _, _ := compute(picks, nil, "", nil, false, nil, 0)
	heavy, _, _ := compute(picks, nil, "", nil, false, &dto.ScoringParams{Weights: map[dto.Dimension]float64{dto.DimensionTech: 10}}, 0)
	if heavy <= base {
		t.Errorf("score with tech weight 10 = %d, want above the default's %d", heavy, base)
	}
}
