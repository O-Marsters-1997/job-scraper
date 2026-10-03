package scoring

import (
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
