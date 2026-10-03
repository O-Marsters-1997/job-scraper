package scoring

import "testing"

func TestParseSalary(t *testing.T) {
	tests := []struct {
		raw        string
		wantAmount int
		wantOK     bool
	}{
		{"£60,000 - £80,000", 80000, true},
		{"$120k", 120000, true},
		{"£7,000", 0, false},
		{"£40.00/hr - £60.00/hr", 0, false},
		{"£50 per hour", 0, false},
		{"£50 hourly", 0, false},
		{"£500 per day", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			amount, _, ok := parseSalary(tt.raw)
			if ok != tt.wantOK || amount != tt.wantAmount {
				t.Errorf("parseSalary(%q) = %d, %v, want %d, %v", tt.raw, amount, ok, tt.wantAmount, tt.wantOK)
			}
		})
	}
}
