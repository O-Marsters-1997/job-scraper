package location_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/location"
)

func TestIsUK(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want bool
	}{
		{"United Kingdom", true},
		{"UK", true},
		{"England", true},
		{"Scotland", true},
		{"Wales", true},
		{"Northern Ireland", true},
		{"London", true},
		{"London, UK", true},
		{"Cambridge, England", true},
		{"Manchester", true},
		{"Edinburgh, Scotland", true},
		{"Remote - UK", true},
		{"Remote (United Kingdom)", true},
		{"London, GB", true},
		{"london, uk", true},
		{"Reading, Berkshire", true},
		{"Birmingham, Alabama", false},
		{"Oxford, Mississippi", false},
		{"Newcastle, NSW", false},
		{"Newcastle, Maine", false},
		{"Cambridge, MA", false},
		{"Cambridge, Massachusetts", false},
		{"London, ON", false},
		{"London, Ontario", false},
		{"New South Wales", false},
		{"Sydney, New South Wales, Australia", false},
		{"Reading, PA", false},
		{"Remote", false},
		{"Remote - US", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := location.IsUK(tt.in); got != tt.want {
				t.Errorf("IsUK(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
