package suitability

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func nicePick(dim dto.Dimension, key string, pYes, pNo, pNotStated float64) evaluatedPick {
	return evaluatedPick{dimension: dim, key: key, label: key, stance: "nice", known: true,
		answer: dto.Answer{PYes: pYes, PNo: pNo, PNotStated: pNotStated}}
}

func avoidPick(key string, pYes, pNo, pNotStated float64) evaluatedPick {
	return evaluatedPick{dimension: "tech", key: key, label: key, stance: "avoid", known: true,
		answer: dto.Answer{PYes: pYes, PNo: pNo, PNotStated: pNotStated}}
}

func TestCompute(t *testing.T) {
	tests := []struct {
		name      string
		picks     []evaluatedPick
		wantScore int
		wantRows  []dto.ScoreRow // Resolved/Effect only, matched by index
	}{
		{
			name:      "nothing known scores 50",
			picks:     nil,
			wantScore: 50,
		},
		{
			name:      "one nice match scores 63",
			picks:     []evaluatedPick{nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05)},
			wantScore: 63,
			wantRows:  []dto.ScoreRow{{Resolved: "yes", Effect: "meets"}},
		},
		{
			name: "four nice dimensions all matched scores 79",
			picks: []evaluatedPick{
				nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05),
				nicePick(dto.DimensionRole, "role:backend", 0.9, 0.05, 0.05),
				nicePick(dto.DimensionSeniority, "seniority:senior", 0.9, 0.05, 0.05),
				nicePick(dto.DimensionWork, "work:remote", 0.9, 0.05, 0.05),
			},
			wantScore: 79,
		},
		{
			name: "two avoids hit, nothing else known scores 21",
			picks: []evaluatedPick{
				avoidPick("tech:java", 0.9, 0.05, 0.05),
				avoidPick("tech:php", 0.9, 0.05, 0.05),
			},
			wantScore: 21,
		},
		{
			name: "two nice picks in one dimension with one matched counted once",
			picks: []evaluatedPick{
				nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05),
				nicePick(dto.DimensionTech, "tech:rust", 0.05, 0.9, 0.05),
			},
			wantScore: 63,
			wantRows: []dto.ScoreRow{
				{Resolved: "yes", Effect: "meets"},
				{Resolved: "no", Effect: "misses"},
			},
		},
		{
			name:      "nice pick with every answer unknown is excluded",
			picks:     []evaluatedPick{nicePick(dto.DimensionTech, "tech:go", 0.2, 0.2, 0.6)},
			wantScore: 50,
			wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
		},
		{
			name:      "an avoid the job lacks is neutral",
			picks:     []evaluatedPick{avoidPick("tech:java", 0.05, 0.9, 0.05)},
			wantScore: 50,
			wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
		},
		{
			name:      "a top probability below 0.6 resolves unknown",
			picks:     []evaluatedPick{nicePick(dto.DimensionTech, "tech:go", 0.55, 0.35, 0.1)},
			wantScore: 50,
			wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, rows := compute(tt.picks)
			if score != tt.wantScore {
				t.Errorf("score = %d, want %d", score, tt.wantScore)
			}
			for i, want := range tt.wantRows {
				if i >= len(rows) {
					t.Fatalf("rows[%d] missing, want %+v", i, want)
				}
				if rows[i].Resolved != want.Resolved || rows[i].Effect != want.Effect {
					t.Errorf("rows[%d] = {Resolved: %q, Effect: %q}, want {Resolved: %q, Effect: %q}",
						i, rows[i].Resolved, rows[i].Effect, want.Resolved, want.Effect)
				}
			}
		})
	}
}

func TestQuestionHash(t *testing.T) {
	a := questionHash("Does the role use Go?")
	b := questionHash("Does the role use Go?")
	c := questionHash("Does the role use Rust?")
	if a != b {
		t.Error("questionHash is not deterministic")
	}
	if a == c {
		t.Error("questionHash collided for different questions")
	}
}
