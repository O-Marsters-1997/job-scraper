package sources_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func TestSalaryRange(t *testing.T) {
	tests := []struct {
		name     string
		min, max int
		currency string
		period   string
		want     string
	}{
		{"both ends", 50000, 70000, "GBP", "year", "GBP 50000 - 70000 / year"},
		{"min only", 50000, 0, "GBP", "year", "GBP 50000+ / year"},
		{"max only", 0, 70000, "GBP", "year", "GBP up to 70000 / year"},
		{"both zero", 0, 0, "GBP", "year", ""},
		{"no currency", 50000, 70000, "", "year", "50000 - 70000 / year"},
		{"no period", 50000, 70000, "USD", "", "USD 50000 - 70000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sources.SalaryRange(tt.min, tt.max, tt.currency, tt.period); got != tt.want {
				t.Errorf("SalaryRange(%d, %d, %q, %q) = %q, want %q", tt.min, tt.max, tt.currency, tt.period, got, tt.want)
			}
		})
	}
}
