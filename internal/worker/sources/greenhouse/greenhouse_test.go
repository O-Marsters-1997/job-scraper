package greenhouse_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "board_acme.json", greenhouse.New("acme"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "board_acme.json", greenhouse.New("acme")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "board_acme.json", got, want)
	}
}

func TestFetchPage_GoldenPayTransparency(t *testing.T) {
	sourcetest.RunGolden(t, "board_reddit.json", greenhouse.New("reddit"))
}
