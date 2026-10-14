package salary_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/salary"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		lo, hi float64
		ok     bool
	}{
		{"words range", "£80,000 to £95,000", 80000, 95000, true},
		{"k range", "£80k - £95k", 80000, 95000, true},
		{"up to", "Up to £90k", 90000, 90000, true},
		{"single dollar", "$120,000", 120000, 120000, true},
		{"en dash euro", "€50k–€70k", 50000, 70000, true},
		{"decimal k", "£62.5k", 62500, 62500, true},
		{"uppercase K", "£80K-£95K", 80000, 95000, true},
		{"third number ignored", "£10 - £20 - £30", 10, 20, true},
		{"empty", "", 0, 0, false},
		{"competitive", "competitive salary", 0, 0, false},
		{"unparseable", "TBC", 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi, ok := salary.Parse(tc.raw)
			if ok != tc.ok || lo != tc.lo || hi != tc.hi {
				t.Errorf("Parse(%q) = %v, %v, %v; want %v, %v, %v", tc.raw, lo, hi, ok, tc.lo, tc.hi, tc.ok)
			}
		})
	}
}
