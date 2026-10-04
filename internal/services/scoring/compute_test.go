package scoring

import (
	"fmt"
	"slices"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestFavouriteLift(t *testing.T) {
	for _, tt := range []struct {
		name        string
		score       int
		wantAtLeast int
		wantAtMost  int
	}{
		{"fair gains most", 50, 59, 61},
		{"great gains less", 85, 88, 90},
		{"poor near zero moves at most 2", 0, 0, 2},
		{"top never drops", 100, 100, 100},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := favouriteLift(tt.score)
			if got < tt.wantAtLeast || got > tt.wantAtMost {
				t.Errorf("favouriteLift(%d) = %d, want %d..%d", tt.score, got, tt.wantAtLeast, tt.wantAtMost)
			}
		})
	}
}

func TestFavouriteLiftShrinksTowardsTop(t *testing.T) {
	if fair, great := favouriteLift(50)-50, favouriteLift(85)-85; fair <= great {
		t.Errorf("fair lift %d, great lift %d, want fair > great", fair, great)
	}
}

func techPicks(stance string, n int) []evaluatedPick {
	picks := make([]evaluatedPick, n)
	for i := range picks {
		picks[i] = evaluatedPick{
			dimension: dto.DimensionTech, key: fmt.Sprintf("tech:%d", i), stance: stance,
			answer: dto.Answer{PYes: 1}, known: true,
		}
	}
	return picks
}

func TestComputeOkStance(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ok, nice []evaluatedPick
	}{
		{"two ok matches credit like one nice", techPicks("ok", 2), techPicks("nice", 1)},
		{"four ok matches credit like two nice", techPicks("ok", 4), techPicks("nice", 2)},
		{"six ok matches saturate like three nice", techPicks("ok", 6), techPicks("nice", 3)},
		{"eight ok matches cap at saturation", techPicks("ok", 8), techPicks("nice", 3)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			okScore, _, _ := compute(tt.ok, nil, "", nil, false)
			niceScore, _, _ := compute(tt.nice, nil, "", nil, false)
			if okScore != niceScore {
				t.Errorf("ok score = %d, want %d (the nice score)", okScore, niceScore)
			}
		})
	}

	t.Run("one ok match scores below one nice match", func(t *testing.T) {
		okScore, _, _ := compute(techPicks("ok", 1), nil, "", nil, false)
		niceScore, _, _ := compute(techPicks("nice", 1), nil, "", nil, false)
		if okScore >= niceScore {
			t.Errorf("ok score = %d, want below %d", okScore, niceScore)
		}
	})

	t.Run("ok pick resolving no still gates", func(t *testing.T) {
		mid := evaluatedPick{
			dimension: dto.DimensionSeniority, key: "seniority:mid", stance: "ok",
			answer: dto.Answer{PNo: 1}, known: true,
		}
		senior := evaluatedPick{
			dimension: dto.DimensionSeniority, key: "seniority:senior",
			answer: dto.Answer{PYes: 1}, known: true,
		}
		score, _, rows := compute([]evaluatedPick{mid}, []evaluatedPick{senior}, "", nil, false)
		if score > gateCap {
			t.Errorf("score = %d, want at most %d", score, gateCap)
		}
		if rows[0].Effect != "gated" {
			t.Errorf("row effect = %q, want gated", rows[0].Effect)
		}
	})
}

func nicePick(dim dto.Dimension, key string, answer dto.Answer, known bool) evaluatedPick {
	return evaluatedPick{dimension: dim, key: key, stance: "nice", answer: answer, known: known}
}

func TestComputeMissingEvidence(t *testing.T) {
	yes := dto.Answer{PYes: 1}
	matched := append(techPicks("nice", 3),
		nicePick(dto.DimensionRole, "role:backend", yes, true),
		nicePick(dto.DimensionSeniority, "seniority:senior", yes, true),
		nicePick(dto.DimensionWork, "work:remote", yes, true),
	)
	silent := []evaluatedPick{
		nicePick(dto.DimensionDomain, "domain:fintech", dto.Answer{PNotStated: 1}, true),
		nicePick(dto.DimensionStage, "stage:seed", dto.Answer{}, false),
		nicePick(dto.DimensionSize, "size:small", dto.Answer{}, false),
	}
	unpicked := []evaluatedPick{
		nicePick(dto.DimensionDomain, "domain:gaming", yes, true),
		nicePick(dto.DimensionEmployment, "employment:contract", yes, true),
	}

	for _, tt := range []struct {
		name     string
		picks    []evaluatedPick
		unpicked []evaluatedPick
		want     int
	}{
		{"sparse ad pulls toward 50", slices.Concat(matched, silent), nil, 83},
		{"full evidence scores as before", matched, nil, 95},
		{"unpicked dimensions never count", slices.Concat(matched, silent), unpicked, 83},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got, _, _ := compute(tt.picks, tt.unpicked, "", nil, false); got != tt.want {
				t.Errorf("compute() score = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestComputeOfficeDaysAvoid(t *testing.T) {
	if w := dimensionSpecs[dto.DimensionWork].Weight; w != 2 {
		t.Fatalf("work weight = %v, want 2", w)
	}
	hybrid := nicePick(dto.DimensionWork, "work:hybrid", dto.Answer{PYes: 1}, true)
	unpicked := []evaluatedPick{
		nicePick(dto.DimensionWork, "work:remote", dto.Answer{PNo: 1}, true),
		nicePick(dto.DimensionWork, "work:onsite", dto.Answer{PNo: 1}, true),
	}
	score := func(officeDays dto.Answer) int {
		office := evaluatedPick{
			dimension: dto.DimensionWork, key: "work:office_3plus", stance: "avoid",
			answer: officeDays, known: true,
		}
		got, _, _ := compute([]evaluatedPick{hybrid, office}, unpicked, "", nil, false)
		return got
	}

	light := score(dto.Answer{PNo: 1})
	heavy := score(dto.Answer{PYes: 1})
	if heavy >= light {
		t.Errorf("3+ office days score = %d, want below the 1-2 day score %d", heavy, light)
	}
	if heavy <= gateCap {
		t.Errorf("3+ office days score = %d, want above the gate cap %d", heavy, gateCap)
	}
	if unstated := score(dto.Answer{PNotStated: 1}); unstated != light {
		t.Errorf("unstated office days score = %d, want %d (the 1-2 day score)", unstated, light)
	}
}

func TestComputeFavourite(t *testing.T) {
	nice := evaluatedPick{
		dimension: dto.Dimension("tech"), key: "tech:go", label: "Go", stance: "nice",
		answer: dto.Answer{PYes: 1}, known: true,
	}
	block := evaluatedPick{
		dimension: dto.Dimension("tech"), key: "tech:php", label: "PHP", stance: "block",
		answer: dto.Answer{PYes: 1}, known: true,
	}
	salaryFloor := &dto.Money{Amount: 100000, Currency: "USD"}

	t.Run("favourite lifts and adds its row", func(t *testing.T) {
		base, _, _ := compute([]evaluatedPick{nice}, nil, "", nil, false)
		score, _, rows := compute([]evaluatedPick{nice}, nil, "", nil, true)
		if score <= base {
			t.Errorf("favourite score = %d, want above %d", score, base)
		}
		last := rows[len(rows)-1]
		if last.Key != favouriteKey || last.Effect != "favourite" {
			t.Errorf("last row = %+v, want the favourite row", last)
		}
	})

	t.Run("not favourite leaves rows untouched", func(t *testing.T) {
		_, _, rows := compute([]evaluatedPick{nice}, nil, "", nil, false)
		for _, r := range rows {
			if r.Effect == "favourite" {
				t.Errorf("unexpected favourite row %+v", r)
			}
		}
	})

	t.Run("blocked favourite stays zero without the row", func(t *testing.T) {
		score, _, rows := compute([]evaluatedPick{nice, block}, nil, "", nil, true)
		if score != 0 {
			t.Errorf("score = %d, want 0", score)
		}
		for _, r := range rows {
			if r.Effect == "favourite" {
				t.Errorf("unexpected favourite row %+v", r)
			}
		}
	})

	t.Run("gated favourite caps at gate", func(t *testing.T) {
		score, _, _ := compute([]evaluatedPick{nice}, nil, "$50000 a year", salaryFloor, true)
		if score > gateCap {
			t.Errorf("score = %d, want at most %d", score, gateCap)
		}
	})
}
