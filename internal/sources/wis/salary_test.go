package wis_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sources/wis"
)

func TestParseSalaryRaw(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "range with 'to'",
			input: "Competitive salary of £80,000 to £95,000 depending on experience.",
			want:  "£80,000 to £95,000",
		},
		{
			name:  "range with hyphen",
			input: "Salary: £60,000 - £75,000",
			want:  "£60,000 - £75,000",
		},
		{
			name:  "range with en-dash",
			input: "Base pay £50,000 – £65,000",
			want:  "£50,000 – £65,000",
		},
		{
			name:  "single amount",
			input: "Up to £120,000 per year",
			want:  "£120,000",
		},
		{
			name:  "dollar",
			input: "Compensation: $100,000 to $140,000",
			want:  "$100,000 to $140,000",
		},
		{
			name:  "euro",
			input: "We offer €70,000 – €90,000",
			want:  "€70,000 – €90,000",
		},
		{
			name:  "no salary",
			input: "Competitive package based on experience.",
			want:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wis.ParseSalaryRaw(tc.input)
			if got != tc.want {
				t.Errorf("ParseSalaryRaw(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}
