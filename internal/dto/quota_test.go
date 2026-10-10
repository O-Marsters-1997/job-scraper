package dto_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestQuotaPercent(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	tests := []struct {
		name        string
		used, limit *float64
		wantPercent *float64
		wantLevel   dto.UsageLevel
	}{
		{"79 percent is ok", f(79), f(100), f(79), dto.UsageOK},
		{"80 percent warns", f(80), f(100), f(80), dto.UsageWarn},
		{"94 percent warns", f(94), f(100), f(94), dto.UsageWarn},
		{"95 percent is critical", f(95), f(100), f(95), dto.UsageCritical},
		{"over limit is critical", f(120), f(100), f(120), dto.UsageCritical},
		{"missing used stays ok", nil, f(100), nil, dto.UsageOK},
		{"missing limit stays ok", f(50), nil, nil, dto.UsageOK},
		{"zero limit stays ok", f(5), f(0), nil, dto.UsageOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percent, level := dto.QuotaPercent(tt.used, tt.limit)
			if level != tt.wantLevel {
				t.Errorf("QuotaPercent level = %q, want %q", level, tt.wantLevel)
			}
			switch {
			case percent == nil && tt.wantPercent == nil:
			case percent == nil || tt.wantPercent == nil:
				t.Errorf("QuotaPercent percent = %v, want %v", percent, tt.wantPercent)
			case *percent != *tt.wantPercent:
				t.Errorf("QuotaPercent percent = %v, want %v", *percent, *tt.wantPercent)
			}
		})
	}
}
