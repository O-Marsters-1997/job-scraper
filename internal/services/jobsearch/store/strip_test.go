package store_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

func TestStripTrackingParams(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"removes tracking params", "utm_source=a&gh_src=b&lever-source=c&lever-origin=d&ref=e&utm_medium=f", ""},
		{"keeps identifying and unknown params in order", "team=x&gh_jid=1&utm_campaign=q&ashby_jid=2&other=3", "team=x&gh_jid=1&ashby_jid=2&other=3"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := store.StripTrackingParams(tt.in); got != tt.want {
				t.Errorf("StripTrackingParams(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
