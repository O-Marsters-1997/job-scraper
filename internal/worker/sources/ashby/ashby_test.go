package ashby_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/ashby"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "board_askdragonfly.json", ashby.New("askdragonfly"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "board_askdragonfly.json", ashby.New("askdragonfly")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "board_askdragonfly.json", got, want)
	}
}
