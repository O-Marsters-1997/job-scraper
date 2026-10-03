package scoring_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

func TestCompanyFactAnswers_SizeBuckets(t *testing.T) {
	tests := []struct {
		size string
		want string
	}{
		{"11-50", "size:startup"},
		{"51-200", "size:scaleup"},
		{"201-500", "size:scaleup"},
		{"501-1,000", "size:large"},
		{"1001-5000", "size:large"},
		{"5000+ employees", "size:enterprise"},
		{"51-200 (2023)", "size:scaleup"},
	}
	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			got := scoring.CompanyFactAnswers(dto.CompanyProfile{Size: tt.size})
			if got[tt.want].PYes != 1 {
				t.Errorf("CompanyFactAnswers(%q) = %+v, want %s yes", tt.size, got, tt.want)
			}
		})
	}

	t.Run("text without a number answers nothing", func(t *testing.T) {
		if got := scoring.CompanyFactAnswers(dto.CompanyProfile{Size: "unknown"}); len(got) != 0 {
			t.Errorf("CompanyFactAnswers(unknown) = %+v, want none", got)
		}
	})
}
