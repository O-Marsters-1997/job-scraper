package scraper_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
)

func TestVerifyBoardRejectsUnsupportedBoards(t *testing.T) {
	for _, pair := range [][2]string{{"missing", "acme"}, {"remoteok", "acme"}, {"greenhouse", ""}, {"greenhouse", "a/b"}} {
		if _, err := scraper.VerifyBoard(t.Context(), pair[0], pair[1]); err == nil {
			t.Errorf("VerifyBoard(%q, %q) = nil, want error", pair[0], pair[1])
		}
	}
}
