package scoring

import "testing"

func TestParseSalary(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantAmount   int
		wantCurrency string
		wantOK       bool
	}{
		{
			name:         "pound range with k suffix takes upper bound",
			raw:          "£55k–70k",
			wantAmount:   70000,
			wantCurrency: "GBP",
			wantOK:       true,
		},
		{
			name:         "thousands separators with currency code and per annum",
			raw:          "55,000 - 70,000 GBP per annum",
			wantAmount:   70000,
			wantCurrency: "GBP",
			wantOK:       true,
		},
		{
			name:         "dollar sign with k suffix",
			raw:          "$120k",
			wantAmount:   120000,
			wantCurrency: "USD",
			wantOK:       true,
		},
		{
			name:   "day rate is not an annual salary",
			raw:    "£600 per day",
			wantOK: false,
		},
		{
			name:   "competitive has no amount",
			raw:    "Competitive",
			wantOK: false,
		},
		{
			name:   "empty string",
			raw:    "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, currency, ok := parseSalary(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if amount != tt.wantAmount || currency != tt.wantCurrency {
				t.Errorf("parseSalary(%q) = (%d, %q), want (%d, %q)",
					tt.raw, amount, currency, tt.wantAmount, tt.wantCurrency)
			}
		})
	}
}
