package eval_test

import (
	"math"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func TestEditBurden(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		output, final string
		want          float64
	}{
		{"identical", "Built the API.", "Built the API.", 0},
		{"case and punctuation ignored", "BUILT, the api", "built the API!", 0},
		{"fully different", "alpha beta gamma", "one two three", 1},
		{"one of four words", "built the fast api", "built the new api", 0.25},
		{"insertion", "built api", "built the api", 1.0 / 3},
		{"both empty", "", "", 0},
		{"empty final", "something", "", 1},
		{"empty output", "", "one two", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eval.EditBurden(tt.output, tt.final); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("EditBurden(%q, %q) = %v, want %v", tt.output, tt.final, got, tt.want)
			}
		})
	}
}
