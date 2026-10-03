package recruitee_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "offers_acme.json", recruitee.New("acme"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "offers_acme.json", recruitee.New("acme")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "offers_acme.json", got, want)
	}
}
