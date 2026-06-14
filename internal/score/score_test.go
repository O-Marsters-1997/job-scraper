package score_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

func TestHeuristicScorer(t *testing.T) {
	scorer := score.NewHeuristicScorer()

	tests := []struct {
		name        string
		card        dto.Job
		cfg         dto.SearchConfig
		wantAtLeast int
		wantAtMost  int
	}{
		{
			name:        "role exact match scores at least 50",
			card:        dto.Job{Title: "Product Engineer", Location: "London"},
			cfg:         dto.SearchConfig{Role: "product engineer"},
			wantAtLeast: 50,
			wantAtMost:  100,
		},
		{
			name:        "no match scores 0",
			card:        dto.Job{Title: "Data Analyst", Location: "Manchester"},
			cfg:         dto.SearchConfig{Role: "software engineer", Location: "London"},
			wantAtLeast: 0,
			wantAtMost:  0,
		},
		{
			name:        "location match adds to score",
			card:        dto.Job{Title: "Unrelated Role", Location: "London"},
			cfg:         dto.SearchConfig{Location: "london"},
			wantAtLeast: 20,
			wantAtMost:  100,
		},
		{
			name: "score clamped to 100",
			card: dto.Job{Title: "Go Engineer go platform", Location: "London"},
			cfg: dto.SearchConfig{
				Role:     "go engineer",
				Keywords: []string{"go", "platform"},
				Location: "london",
			},
			wantAtLeast: 100,
			wantAtMost:  100,
		},
		{
			name:        "partial keyword match adds partial score",
			card:        dto.Job{Title: "Go Developer"},
			cfg:         dto.SearchConfig{Keywords: []string{"go", "platform"}},
			wantAtLeast: 15,
			wantAtMost:  15,
		},
		{
			name:        "empty config scores 0",
			card:        dto.Job{Title: "Senior Software Engineer", Location: "London"},
			cfg:         dto.SearchConfig{},
			wantAtLeast: 0,
			wantAtMost:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scorer.Score(tt.card, tt.cfg)
			if got < tt.wantAtLeast || got > tt.wantAtMost {
				t.Errorf("Score() = %d, want between %d and %d", got, tt.wantAtLeast, tt.wantAtMost)
			}
		})
	}
}
